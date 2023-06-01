package driver

import (
	"github.com/sirupsen/logrus"
	kexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
)

const runningState = "running"

type prodAttachmentValidator struct {
}

type AttachmentValidator interface {
	readFile(name string) ([]byte, error)
	evalSymlinks(path string) (string, error)
}

type volumeStatistics struct {
	availableBytes, totalBytes, usedBytes  int64
	availabInodes, totalInodes, usedInodes int64
}

// Mounter is responsible for formating and mounting volumes
type Mounter interface {
	// Format formats the source with the given filesystem type
	Format(source, fsType string) error

	// Mount mounts source to target with the given fstype and options.
	Mount(source, target, fsType string, options ...string) error

	// Unmount unmounts the given target.
	Unmount(target string) error

	// IsAttached checks whether the source device is in the running state.
	IsAttached(source string) error

	// IsFormatted checks whether the source device is formatted or not. It
	// returns true if the source device is already formatted.
	IsFormatted(source string) (bool, error)

	// IsMounted checks whether the target path is a correct mount(i.e:propagated).
	// It returns true if it's mounted. An error is returned in case of system
	// errors or if it's mounted incorrectly.
	IsMounted(target string) (bool, error)

	GetDeviceName(mounter mount.Interface, mountPath string) (string, error)

	// GetStatistics returns capacity-related volume statistics for the given volume path.
	GetStatistics(volumePath string) (volumeStatistics, error)

	// IsBlockDevice checks whether the device at the path is a block device
	IsBlockDevice(volumePath string) (bool, error)
}

type mounter struct {
	log                 *logrus.Entry
	kMounter            *mount.SafeFormatAndMount
	attachmentValidator AttachmentValidator
}

func newMounter(log *logrus.Entry) *mounter {
	kMounter := &mount.SafeFormatAndMount{
		Interface: mount.New(""),
		Exec:      kexec.New(),
	}

	return &mounter{
		kMounter:            kMounter,
		log:                 log,
		attachmentValidator: &prodAttachmentValidator{},
	}
}
