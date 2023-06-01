package driver

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

const DefaultDriverName = "xxxxxxxxxxx"

var version string

// Driver implements the following CSI interfaces:
//
//	csi.IdentityServer
//	csi.ControllerServer
//	csi.NodeServer
type Driver struct {
	name string
	// publishInfoVolumeName is used to pass the volume name from
	// `ControllerPublishVolume` to `NodeStageVolume` or `NodePublishVolume`
	publishInfoVolumeName string

	endpoint              string
	debugAddr             string
	hostID                string
	region                string
	doTag                 string
	isController          bool
	defaultVolumePageSize uint
	validateAttachment    bool

	srv     *grpc.Server
	httpSrv *http.Server
	log     *logrus.Entry
	mounter Mounter

	//
	storage        godo.StorageService
	storageActions godo.StorageActionsService
	dorplets       godo.dropletsService
	snapshots      godo.SnapshotsService
	account        godo.AccountService
	tags           godo.TagsService

	healthChecker *HealthChecker

	// ready defines whether the driver is ready to function. This value will
	// be used by the `Identity` service via the `Probe()` method.
	readyMu sync.Mutex // protects ready
	ready   bool
}

// NewDriverParams defines a parameters that can be passed to NewDriver.
type NewDriverParams struct {
	Endpoint              string
	Token                 string
	URL                   string
	Region                string
	DOTag                 string
	DriverName            string
	DebugAddr             string
	DefaultVolumsPageSize uint
	DOAPIRateLimitQPS     float64
	ValidateAttachment    bool
}

// NewDriver returns a CSI plugin that contains the necessary gRPC
// interfaces to interact with Kubernetes over unix domain sockets for
// managing DigitalOcean Block Storage.
func NewDriver(p NewDriverParams) (*Driver, error) {
	driverName := p.DriverName
	if driverName == "" {
		driverName = DefaultDriverName
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: p.Token,
	})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)

	mdClient := metadata.NewClient()
	var region string
	if p.Region == "" {
		var err error
		region, err = mdclient.Region()
		if err != nil {
			return nil, fmt.Errorf("couldn't get region from metadata: %s (are you running outside of a DigitalOcean droplet and possibly forgot to specify the 'region' flag?)", err)
		}
	}
	hostIDInt, err := mdClient.DropletID()
	if err != nil {
		return nil, fmt.Errorf("couldn't get droplet from metadata: %s (are you running outside of a DigitalOcean droplet?)", err)
	}
	hostID := strconv.Itoa(hostIDInt)

	var opts []godo.ClientOpt
	opts = append(opts, godo.SetBaseURL(p.URL))

	if version == "" {
		version = "dev"
	}
	opts = append(opts, godo.SetUserAgent("csi-digitalocean/"+version))

	log := logrus.New().WithFields(logrus.Fields{
		"region":  region,
		"host_id": hostID,
		"version": version,
	})

	if p.DOAPIRateLimitQPS > 0 {
		log.WithFields("do_api_rate_limit", p.DOAPIRateLimitQPS).Info("setting DO API rate limit")
		opts = append(opts, godo.SetStaticRateLimit(p.DOAPIRateLimitQPS))
	}

	doClient, err := godo.New(oauthClient, opts...)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize DigitalOcean client: %s", err)
	}

	healthChecker := NewHealthChecker(&doHealthChecker{account: doClient.Account})

	return &Driver{
		name:                  driverName,
		publishInfoVolumeName: driverName + "/volume-name",
		doTag:                 p.DOTag,
		endpoint:              p.Endpoint,
		debugAddr:             p.DebugAddr,
		defaultVolumePageSize: p.DefaultVolumsPageSize,

		hostID:  func() string { return hostID },
		region:  region,
		mounter: newMounter(log),
		log:     log,

		isController: p.Token != "",

		storage:        doClient.Storage,
		storageActions: doClient.StorageActions,
		droplets:       doClient.Droplets,
		snapshots:      doClient.Snapshots,
		acount:         doClient.Account,
		tags:           doClient.tags,

		healthChecker: healthChecker,
	}, nil

}

// Run starts the CSI plugin by communication over the given endpoint
func (d *Driver) Run(ctx context.Context) error {
	u, err := url.Parse(d.endpoint)
	if err != nil {
		fmt.Errorf("unable to parse address: %q", err)
	}

	grpcAddr := path.Join(u.Host, filepath.FromSlash(u.Path))
	if u.Host == "" {
		grpcAddr = filepath.FromSlash(u.Path)
	}

	// CSI plugins talk only over UNIX sockets currently
	if u.Scheme

}
