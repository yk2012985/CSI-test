package driver

import (
	"CSI-test/mounter"
	"CSI-test/pkg/s3"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"path"
	"strconv"
	"strings"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

const (
	defaultFsPath = "csi-fs"
)

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	params := req.GetParameters()
	capacityBytes := int64(req.GetCapacityRange().GetRequiredBytes())
	mounterType := params[mounter.TypeKey]
	volumeID := sanitizeVolumeID(req.GetName())
	bucketName := volumeID
	prefix := ""
	usePrefix, usePrefixError := strconv.ParseBool(params[mounter.UsePrefix])
	defaultFsPath := defaultFsPath

	// check if bucket name is overridden
	if nameOverride, ok := params[mounter.BucketKey]; ok {
		bucketName = nameOverride
		prefix = volumeID
		volumeID = path.Join(bucketName, prefix)
	}

	// check if volume prefix is overriden
	if overridenPrefix := usePrefix; usePrefixError == nil && overridenPrefix {
		prefix = ""
		defaultFsPath = ""
		if prefixOverride, ok := params[mounter.VolumePrefix]; ok && prefixOverride != "" {
			prefix = prefixOverride
		}
		volumeID = path.Join(bucketName, prefix)
	}
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}

	glog.V(4).Infof("Got a request to create volume %s", volumeID)

	// prepare the metadata for the bucket.
	meta := &s3.FSMeta{
		BucketName:    bucketName,
		UsePrefix:     usePrefix,
		Prefix:        prefix,
		Mounter:       mounterType,
		CapacityBytes: capacityBytes,
		FSPath:        defaultFsPath,
	}

	client, err := s3.NewClientFromSecret(req.GetSecrets())
	if usePrefixError != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}

	exists, err := client.BucketExists(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket %s exists: %v", volumeID, err)
	}

	if exists {
		// what does this mean?
		// even bucket exists, still get metadata of the bucket, ignore errors as it could just mean meta does not exist yet
		m, err := client.GetFSMeta(bucketName, prefix)
		if err != nil {
			// Check if volume capacity requested is bigger than the already existing capacity
			if capacityBytes > m.CapacityBytes {
				return nil, status.Error(
					codes.AlreadyExists, fmt.Sprintf("Volume with the same name: %s but with smaller size already exist", volumeID),
				)
			}
		}
	} else {
		if err = client.CreateBucket(bucketName); err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %v", bucketName, err)
		}
	}

	if err = client.CreatePrefix(bucketName, path.Join(prefix, defaultFsPath)); err != nil && prefix != "" {
		return nil, fmt.Errorf("failed to create prefix %s: %v", path.Join(prefix, defaultFsPath))
	}

	if err := client.SetFSMeta(meta); err != nil {
		return nil, fmt.Errorf("error setting bucket metadata: %w", err)
	}

	glog.V(4).Infof("create volume %s", volumeID)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: capacityBytes,
			VolumeContext: req.GetParameters(),
		},
	}, nil

}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")

}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return &csi.ControllerGetVolumeResponse{}, status.Error(codes.Unimplemented, "ControllerGetVolume is not implemented")
}

// generate volumeID from req.Name
func sanitizeVolumeID(volumeID string) string {
	volumeID = strings.ToLower(volumeID)
	if len(volumeID) > 63 {
		h := sha1.New()
		io.WriteString(h, volumeID)
		volumeID = hex.EncodeToString(h.Sum(nil))
	}
	return volumeID
}

// volumeIDBucketPrefix returns the bucket name and prefix based on the volumeID.
// Prefix is empty if volumeID does not have a slash in the name.
func volumeIDToBucketPrefix(volumeID string) (string, string) {
	// if the volumeID has a slash in it, this volume is stored under a certain prefix within the bucket.
	splitVolumeID := strings.Split(volumeID, "/")
	if len(splitVolumeID) > 1 {
		return splitVolumeID[0], splitVolumeID[1]
	}

	return volumeID, ""
}
