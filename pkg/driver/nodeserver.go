package driver

import csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"

type nodeServer struct {
	*csicommon.DefaultNodeServer
}
