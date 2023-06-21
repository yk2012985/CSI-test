package mounter

/*
Mounter interface which can be implemented by the different mounter types
*/
type Mounter interface {
	Stage(stagePath string) error
	Unstage(stagePath string) error
	Mount(source string, target string) error
}

const (
	s3fsMounterType     = "s3fs"
	goofysMounterType   = "goofys"
	s3backerMounterType = "s3backer"
	rcloneMounterType   = "rclone"
	TypeKey             = "mounter"
	BucketKey           = "bucket"
	VolumePrefix        = "prefix"
	UsePrefix           = "usePrefix"
)
