package mounter

import "CSI-test/pkg/s3"

const (
	goofysCmd     = "goofys"
	defaultRegion = "us-east-1"
)

// Inplements Mounter
type goofysMounter struct {
	meta            *s3.FSMeta
	endpoint        string
	region          string
	accessKeyID     string
	secretAccessKey string
}

func newGoofysMounter(meta *s3.FSMeta, cfg *s3.Config) (Mounter, error) {
	region := cfg.Region

}
