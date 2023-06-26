package mounter

import (
	"CSI-test/pkg/s3"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/mount-utils"
	"k8s.io/utils/mount"
	"os"
	"os/exec"
	"time"
)

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

// New returns a new mounter depending on the mounterType parameter
func New(meta *s3.FSMeta, cfg *s3.Config) (Mounter, error) {
	mounter := meta.Mounter
	if len(meta.Mounter) == 0 {
		mounter = cfg.Mounter
	}
	switch mounter {
	case s3fsMounterType:
		return newS3fsMounter(meta, cfg)
	case goofysMounterType:
		return new
	}
}

func fuseMount(path string, command string, args []string) error {
	cmd := exec.Command(command, args...)
	glog.V(3).Infof("Mounting fuse with command: %s and args: %s", command, args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error fuseMount command: %s\nargs: %s\noutput", command, args)
	}
	return waitForMount(path, 10*time.Second)
}

func waitForMount(path string, timeout time.Duration) error {
	var elapsed time.Duration
	var interval = 10 * time.Millisecond
	for {
		notMount, err := mount.New("").IsLikelyNotMountPoint(path)
		if err != nil {
			return err
		}
		if !notMount {
			return nil
		}
		time.Sleep(interval)
		elapsed = elapsed + interval
		if elapsed >= timeout {
			return errors.New("Timeout waiting for mount")
		}
	}
}
