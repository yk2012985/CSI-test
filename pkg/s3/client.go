package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
	"path"
	"strings"
)

const (
	metadataName = ".metadata.json"
)

type s3Client struct {
	Config      *Config
	parastorSvc *s3.S3
	ctx         context.Context
}

// Config holds values to configure the driver
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Endpoint        string
	Mounter         string
}

type FSMeta struct {
	BucketName    string `json:"Name"`
	Prefix        string `json:"Prefix"`
	UsePrefix     bool   `json:"UsePrefix"`
	Mounter       string `json:"Mounter"`
	FSPath        string `json:"FSPath"`
	CapacityBytes int64  `json:"CapacityBytes"`
}

func NewClient(cfg *Config) (*s3Client, error) {
	var client = &s3Client{}

	client.Config = cfg
	//u, err := url.Parse(client.Config.Endpoint)
	//if err != nil {
	//	return nil, err
	//}
	//ssl := u.Scheme == "https"
	//endpoint := u.Hostname()
	//if u.Port() != "" {
	//	endpoint = u.Hostname() + ":" + u.Port()
	//}

	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(client.Config.AccessKeyID, client.Config.SecretAccessKey, ""),
		Endpoint:         aws.String(client.Config.Endpoint),
		Region:           aws.String(client.Config.Region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}
	parastorSvc := s3.New(sess)

	//minioClient, err := minio.New(endpoint, &minio.Options{
	//	Creds:  credentials.NewStaticV4(client.Config.AccessKeyID, client.Config.SecretAccessKey, client.Config.Region),
	//	Secure: ssl,
	//})
	//if err != nil {
	//	return nil, err
	//}
	client.parastorSvc = parastorSvc
	client.ctx = context.Background()
	return client, nil
}

func NewClientFromSecret(secret map[string]string) (*s3Client, error) {
	return NewClient(&Config{
		AccessKeyID:     secret["accessKeyID"],
		SecretAccessKey: secret["secretAccessKey"],
		Region:          secret["region"],
		Endpoint:        secret["endpoint"],
		// Mounter is set in the volume preferences, not secrets
		Mounter: "",
	})
}

func (client *s3Client) BucketExists(bucketName string) (bool, error) {
	_, err := client.parastorSvc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String("testBucketExistKey"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return false, nil
			case s3.ErrCodeNoSuchKey:
				return true, nil
			}
		}
		return false, err

	}
	return true, nil

	//return client.minio.BucketExists(client.ctx, bucketName)
}

func (client *s3Client) CreateBucket(bucketName string) error {
	prarms := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName + "/"),
	}

	_, err := client.parastorSvc.CreateBucket(prarms)
	if err != nil {
		return err
	}
	glog.V(3).Infof("Waiting for bucket %q to be created...\n", bucketName)
	err = client.parastorSvc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName + "/"),
	})
	if err != nil {
		return err
	}
	glog.V(3).Infof("Bucket %q successfully created\n", bucketName)
	return nil
	//return client.minio.MakeBucket(client.ctx, bucketName, minio.MakeBucketOptions{Region: client.Config.Region})
}

// CreatePrefix What does this func do?
func (client *s3Client) CreatePrefix(bucketName string, prefix string) error {
	_, err := client.parastorSvc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String(prefix + "/"),
		Body:   bytes.NewReader([]byte("")),
	})
	if err != nil {
		return err
	}
	return nil
}

func (client *s3Client) SetFSMeta(meta *FSMeta) error {
	content, err := json.Marshal(meta)
	if err != nil {
		glog.V(4).Infof("Json marshal failed: %v\n", err)
		return err
	}
	_, err = client.parastorSvc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(meta.BucketName + "/"),
		Key:         aws.String(path.Join(meta.Prefix, metadataName)),
		ContentType: aws.String("application/json"),
		Body:        strings.NewReader(string(content)),
	})
	return err

}

// GetFSMeta get metadata of bucket
func (client *s3Client) GetFSMeta(bucketName, prefix string) (*FSMeta, error) {
	resp, err := client.parastorSvc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String(path.Join(prefix, metadataName)),
	})
	if err != nil {
		return &FSMeta{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	result := FSMeta{}
	if err := decoder.Decode(&result); err != nil {
		return &FSMeta{}, err
	}
	return &result, nil

}

func (client *s3Client) PutObjectToBucket(bucketName, keyName string) error {
	_, err := client.parastorSvc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String(keyName),
		Body:   strings.NewReader("Expected contents"),
	})
	if err != nil {
		glog.V(4).Infof("There is an error occurred: %v\n.", err)
		return err
	}
	err = client.parastorSvc.WaitUntilObjectExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String(keyName),
	})
	if err != nil {
		glog.V(4).Infof("Can't verify the object exists: %v\n", err)
		return err
	}
	return nil
}

func (client *s3Client) GetObject(bucketName, keyName string, svc *s3.S3) (interface{}, error) {
	gotObject, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName + "/"),
		Key:    aws.String(keyName),
	})
	if err != nil {
		glog.V(3).Infof("Getting object wrong: %v\n", err)
		return nil, err
	}
	return gotObject, nil
}
