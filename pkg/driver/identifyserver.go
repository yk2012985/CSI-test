package driver

import csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"

type identifyServer struct {
	*csicommon.DefaultIdentityServer
}
