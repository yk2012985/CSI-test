package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"net/url"
	"path"
)

const (
	metadataName = ".metadata.json"
)

type s3Client struct {
	Config *Config
	minio  *minio.Client
	ctx    context.Context
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
	u, err := url.Parse(client.Config.Endpoint)
	if err != nil {
		return nil, err
	}
	ssl := u.Scheme == "https"
	endpoint := u.Hostname()
	if u.Port() != "" {
		endpoint = u.Hostname() + ":" + u.Port()
	}
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(client.Config.AccessKeyID, client.Config.SecretAccessKey, client.Config.Region),
		Secure: ssl,
	})
	if err != nil {
		return nil, err
	}
	client.minio = minioClient
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
	return client.minio.BucketExists(client.ctx, bucketName)
}

func (client *s3Client) CreateBucket(bucketName string) error {
	return client.minio.MakeBucket(client.ctx, bucketName, minio.MakeBucketOptions{Region: client.Config.Region})
}

// CreatePrefix What does this func do?
func (client *s3Client) CreatePrefix(bucketName string, prefix string) error {
	_, err := client.minio.PutObject(client.ctx, bucketName, prefix+"/", bytes.NewReader([]byte("")), 0, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (client *s3Client) SetFSMeta(meta *FSMeta) error {
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(meta)
	opts := minio.PutObjectOptions{ContentType: "application/json"}
	_, err := client.minio.PutObject(
		client.ctx, meta.BucketName, path.Join(meta.Prefix, metadataName), b, int64(b.Len()), opts)
	return err

}

func (client *s3Client) GetFSMeta(bucketName, prefix string) (*FSMeta, error) {
	// what does this mean?
	opts := minio.GetObjectOptions{}
	obj, err := client.minio.GetObject(client.ctx, bucketName, path.Join(prefix, metadataName), opts)
	if err != nil {
		return &FSMeta{}, err
	}
	objInfo, err := obj.Stat()
	if err != nil {
		return &FSMeta{}, err
	}
	b := make([]byte, objInfo.Size)
	_, err = obj.Read(b)

	if err != nil && err != io.EOF {
		return &FSMeta{}, err
	}
	var meta FSMeta
	err = json.Unmarshal(b, &meta)
	return &meta, err
}
