package service

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-test/mock/cache"
	"golang.org/x/net/context"

	"github.com/golang/protobuf/ptypes"
)

const (
	// Name is the name of the CSI plug-in.
	Name = "io.kubernetes.storage.mock"

	// VendorVersion is the version returned by GetPluginInfo.
	VendorVersion = "0.3.0"
)

// Manifest is the SP's manifest.
var Manifest = map[string]string{
	"url": "https://github.com/kubernetes-csi/csi-test/mock",
}

type Config struct {
	DisableAttach bool
	DriverName    string
	AttachLimit   int64
}

// Service is the CSI Mock service provider.
type Service interface {
	csi.ControllerServer
	csi.IdentityServer
	csi.NodeServer
}

type service struct {
	sync.Mutex
	nodeID       string
	vols         []csi.Volume
	volsRWL      sync.RWMutex
	volsNID      uint64
	snapshots    cache.SnapshotCache
	snapshotsNID uint64
	config       Config
}

type Volume struct {
	sync.Mutex
	VolumeCSI       csi.Volume
	NodeID          string
	ISStaged        bool
	ISPublished     bool
	StageTargetPath string
	TargetPath      string
}

var MockVolumes map[string]Volume

// New returns a new Service.
func New(config Config) Service {
	s := &service{
		nodeID: config.DriverName,
		config: config,
	}
	s.snapshots = cache.NewSnapshotCache()
	s.vols = []csi.Volume{
		s.newVolume("Mock Volume 1", gib100),
		s.newVolume("Mock Volume 2", gib100),
		s.newVolume("Mock Volume 3", gib100),
	}
	MockVolumes = map[string]Volume{}

	s.snapshots.Add(s.newSnapshot("Mock Snapshot 1", "1", map[string]string{"Description": "snapshot 1"}))
	s.snapshots.Add(s.newSnapshot("Mock Snapshot 2", "2", map[string]string{"Description": "snapshot 2"}))
	s.snapshots.Add(s.newSnapshot("Mock Snapshot 3", "3", map[string]string{"Description": "snapshot 3"}))

	return s
}

const (
	kib    int64 = 1024
	mib    int64 = kib * 1024
	gib    int64 = mib * 1024
	gib100 int64 = gib * 100
	tib    int64 = gib * 1024
	tib100 int64 = tib * 100
)

func (s *service) newVolume(name string, capcity int64) csi.Volume {
	return csi.Volume{
		VolumeId:      fmt.Sprintf("%d", atomic.AddUint64(&s.volsNID, 1)),
		VolumeContext: map[string]string{"name": name},
		CapacityBytes: capcity,
	}
}

func (s *service) findVol(k, v string) (volIdx int, volInfo csi.Volume) {
	s.volsRWL.RLock()
	defer s.volsRWL.RUnlock()
	return s.findVolNoLock(k, v)
}

func (s *service) findVolNoLock(k, v string) (volIdx int, volInfo csi.Volume) {
	volIdx = -1

	for i, vi := range s.vols {
		switch k {
		case "id":
			if strings.EqualFold(v, vi.GetVolumeId()) {
				return i, vi
			}
		case "name":
			if n, ok := vi.VolumeContext["name"]; ok && strings.EqualFold(v, n) {
				return i, vi
			}
		}
	}

	return
}

func (s *service) findVolByName(
	ctx context.Context, name string) (int, csi.Volume) {

	return s.findVol("name", name)
}

func (s *service) newSnapshot(name, sourceVolumeId string, parameters map[string]string) cache.Snapshot {

	ptime := ptypes.TimestampNow()
	return cache.Snapshot{
		Name:       name,
		Parameters: parameters,
		SnapshotCSI: csi.Snapshot{
			SnapshotId:     fmt.Sprintf("%d", atomic.AddUint64(&s.snapshotsNID, 1)),
			CreationTime:   ptime,
			SourceVolumeId: sourceVolumeId,
			ReadyToUse:     true,
		},
	}
}
