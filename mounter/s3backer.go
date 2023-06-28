package mounter

import (
	"CSI-test/pkg/s3"
	"net/url"
	"path"
)

// Implements Mounter
type s3backerMounter struct {
	meta *s3.FSMeta
	url string
	region string
	accessKeyID string
	secretAccessKey string
	ssl bool
}

const (
	s3backerCmd = "s3backer"
	s3backerFsType = "xfs"
	s3backerDevice = "file"
	// blockSize to use in k
	s3backerBlockSize = 1024 * 1024 * 1024 // 1GiB
	// S3backerLoopDevice the loop device required by s3backer
	S3backerLoopDevice = "/dev/loop0"
)

func newS3backerMounter(meta *s3.FSMeta, cfg *s3.Config) (Mounter, error) {
	url, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, meta.BucketName, meta.Prefix, meta.FSPath)
	//s3backer cannot work with 0 size volumes
	if meta.CapacityBytes == 0 {
		meta.CapacityBytes = s3backerBlockSize
	}
	s3backer := &s3backerMounter{
		meta: meta,
		url: cfg.Endpoint,
		region: cfg.Region,
		accessKeyID: cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
		ssl: url.Scheme == "https",
	}
	return s3backer,
}

