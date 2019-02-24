package service

import (
	"fmt"
	"math"
	"path"
	"reflect"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	MaxStorageCapacity = tib
	ReadOnlyKey        = "readonly"
)

func (s *service) CreateVolume(
	ctx context.Context,
	req *csi.CreateVolumeRequest) (
	*csi.CreateVolumeResponse, error) {

	if len(req.Name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume Name cannot be empty")
	}
	if req.VolumeCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities cannot be empty")
	}

	// Check to see if the volume already exists.
	if i, v := s.findVolByName(ctx, req.Name); i >= 0 {
		// Requested volume name already exists, need to check if the existing volume's
		// capacity is more or equal to new request's capacity.
		if v.GetCapacityBytes() < req.GetCapacityRange().GetRequiredBytes() {
			return nil, status.Error(codes.AlreadyExists,
				fmt.Sprintf("Volume with name %s already exists", req.GetName()))
		}
		return &csi.CreateVolumeResponse{Volume: &v}, nil
	}

	// If no capacity is specified then use 100GiB
	capacity := gib100
	if cr := req.CapacityRange; cr != nil {
		if rb := cr.RequiredBytes; rb > 0 {
			capacity = rb
		}
		if lb := cr.LimitBytes; lb > 0 {
			capacity = lb
		}
	}
	// Check for maximum available capacity
	if capacity >= MaxStorageCapacity {
		return nil, status.Errorf(codes.OutOfRange, "Requested capacity %d exceeds maximum allowed %d", capacity, MaxStorageCapacity)
	}
	// Create the volume and add it to the service's in-mem volume slice.
	v := s.newVolume(req.Name, capacity)
	s.volsRWL.Lock()
	defer s.volsRWL.Unlock()
	s.vols = append(s.vols, v)
	MockVolumes[v.GetVolumeId()] = Volume{
		VolumeCSI:       v,
		NodeID:          "",
		ISStaged:        false,
		ISPublished:     false,
		StageTargetPath: "",
		TargetPath:      "",
	}

	return &csi.CreateVolumeResponse{Volume: &v}, nil
}

func (s *service) DeleteVolume(
	ctx context.Context,
	req *csi.DeleteVolumeRequest) (
	*csi.DeleteVolumeResponse, error) {

	s.volsRWL.Lock()
	defer s.volsRWL.Unlock()

	//  If the volume is not specified, return error
	if len(req.VolumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}

	// If the volume does not exist then return an idempotent response.
	i, _ := s.findVolNoLock("id", req.VolumeId)
	if i < 0 {
		return &csi.DeleteVolumeResponse{}, nil
	}

	// This delete logic preserves order and prevents potential memory
	// leaks. The slice's elements may not be pointers, but the structs
	// themselves have fields that are.
	copy(s.vols[i:], s.vols[i+1:])
	s.vols[len(s.vols)-1] = csi.Volume{}
	s.vols = s.vols[:len(s.vols)-1]
	log.WithField("volumeID", req.VolumeId).Debug("mock delete volume")
	return &csi.DeleteVolumeResponse{}, nil
}

func (s *service) ControllerPublishVolume(
	ctx context.Context,
	req *csi.ControllerPublishVolumeRequest) (
	*csi.ControllerPublishVolumeResponse, error) {

	if s.config.DisableAttach {
		return nil, status.Error(codes.Unimplemented, "ControllerPublish is not supported")
	}

	if len(req.VolumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}
	if len(req.NodeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Node ID cannot be empty")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities cannot be empty")
	}

	if req.NodeId != s.nodeID {
		return nil, status.Errorf(codes.NotFound, "Not matching Node ID %s to Mock Node ID %s", req.NodeId, s.nodeID)
	}

	s.volsRWL.Lock()
	defer s.volsRWL.Unlock()

	i, v := s.findVolNoLock("id", req.VolumeId)
	if i < 0 {
		return nil, status.Error(codes.NotFound, req.VolumeId)
	}

	// devPathKey is the key in the volume's attributes that is set to a
	// mock device path if the volume has been published by the controller
	// to the specified node.
	devPathKey := path.Join(req.NodeId, "dev")

	// Check to see if the volume is already published.
	if device := v.VolumeContext[devPathKey]; device != "" {
		var volRo bool
		var roVal string
		if ro, ok := v.VolumeContext[ReadOnlyKey]; ok {
			roVal = ro
		}

		if roVal == "true" {
			volRo = true
		} else {
			volRo = false
		}

		// Check if readonly flag is compatible with the publish request.
		if req.GetReadonly() != volRo {
			return nil, status.Error(codes.AlreadyExists, "Volume published but has incompatible readonly flag")
		}

		return &csi.ControllerPublishVolumeResponse{
			PublishContext: map[string]string{
				"device":   device,
				"readonly": roVal,
			},
		}, nil
	}

	var roVal string
	if req.GetReadonly() {
		roVal = "true"
	} else {
		roVal = "false"
	}

	// Publish the volume.
	device := "/dev/mock"
	v.VolumeContext[devPathKey] = device
	v.VolumeContext[ReadOnlyKey] = roVal
	s.vols[i] = v

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			"device":   device,
			"readonly": roVal,
		},
	}, nil
}

func (s *service) ControllerUnpublishVolume(
	ctx context.Context,
	req *csi.ControllerUnpublishVolumeRequest) (
	*csi.ControllerUnpublishVolumeResponse, error) {

	if s.config.DisableAttach {
		return nil, status.Error(codes.Unimplemented, "ControllerPublish is not supported")
	}

	if len(req.VolumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}
	nodeID := req.NodeId
	if len(nodeID) == 0 {
		// If node id is empty, no failure as per Spec
		nodeID = s.nodeID
	}

	if req.NodeId != s.nodeID {
		return nil, status.Errorf(codes.NotFound, "Node ID %s does not match to expected Node ID %s", req.NodeId, s.nodeID)
	}

	s.volsRWL.Lock()
	defer s.volsRWL.Unlock()

	i, v := s.findVolNoLock("id", req.VolumeId)
	if i < 0 {
		return nil, status.Error(codes.NotFound, req.VolumeId)
	}

	// devPathKey is the key in the volume's attributes that is set to a
	// mock device path if the volume has been published by the controller
	// to the specified node.
	devPathKey := path.Join(nodeID, "dev")

	// Check to see if the volume is already unpublished.
	if v.VolumeContext[devPathKey] == "" {
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	// Unpublish the volume.
	delete(v.VolumeContext, devPathKey)
	delete(v.VolumeContext, ReadOnlyKey)
	s.vols[i] = v

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (s *service) ValidateVolumeCapabilities(
	ctx context.Context,
	req *csi.ValidateVolumeCapabilitiesRequest) (
	*csi.ValidateVolumeCapabilitiesResponse, error) {

	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}
	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, req.VolumeId)
	}
	i, _ := s.findVolNoLock("id", req.VolumeId)
	if i < 0 {
		return nil, status.Error(codes.NotFound, req.VolumeId)
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters(),
		},
	}, nil
}

func (s *service) ListVolumes(
	ctx context.Context,
	req *csi.ListVolumesRequest) (
	*csi.ListVolumesResponse, error) {

	// Copy the mock volumes into a new slice in order to avoid
	// locking the service's volume slice for the duration of the
	// ListVolumes RPC.
	var vols []csi.Volume
	func() {
		s.volsRWL.RLock()
		defer s.volsRWL.RUnlock()
		vols = make([]csi.Volume, len(s.vols))
		copy(vols, s.vols)
	}()

	var (
		ulenVols      = int32(len(vols))
		maxEntries    = req.MaxEntries
		startingToken int32
	)

	if v := req.StartingToken; v != "" {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"startingToken=%d !< int32=%d",
				startingToken, math.MaxUint32)
		}
		startingToken = int32(i)
	}

	if startingToken > ulenVols {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"startingToken=%d > len(vols)=%d",
			startingToken, ulenVols)
	}

	// Discern the number of remaining entries.
	rem := ulenVols - startingToken

	// If maxEntries is 0 or greater than the number of remaining entries then
	// set maxEntries to the number of remaining entries.
	if maxEntries == 0 || maxEntries > rem {
		maxEntries = rem
	}

	var (
		i       int
		j       = startingToken
		entries = make(
			[]*csi.ListVolumesResponse_Entry,
			maxEntries)
	)

	for i = 0; i < len(entries); i++ {
		entries[i] = &csi.ListVolumesResponse_Entry{
			Volume: &vols[j],
		}
		j++
	}

	var nextToken string
	if n := startingToken + int32(i); n < ulenVols {
		nextToken = fmt.Sprintf("%d", n)
	}

	return &csi.ListVolumesResponse{
		Entries:   entries,
		NextToken: nextToken,
	}, nil
}

func (s *service) GetCapacity(
	ctx context.Context,
	req *csi.GetCapacityRequest) (
	*csi.GetCapacityResponse, error) {

	return &csi.GetCapacityResponse{
		AvailableCapacity: MaxStorageCapacity,
	}, nil
}

func (s *service) ControllerGetCapabilities(
	ctx context.Context,
	req *csi.ControllerGetCapabilitiesRequest) (
	*csi.ControllerGetCapabilitiesResponse, error) {

	caps := []*csi.ControllerServiceCapability{
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_GET_CAPACITY,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
				},
			},
		},
	}

	if !s.config.DisableAttach {
		caps = append(caps, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
				},
			},
		})
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: caps,
	}, nil
}

func (s *service) CreateSnapshot(ctx context.Context,
	req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	// Check arguments
	if len(req.GetName()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot Name cannot be empty")
	}
	if len(req.GetSourceVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot SourceVolumeId cannot be empty")
	}

	// Check to see if the snapshot already exists.
	if i, v := s.snapshots.FindSnapshot("name", req.GetName()); i >= 0 {
		// Requested snapshot name already exists
		if v.SnapshotCSI.GetSourceVolumeId() != req.GetSourceVolumeId() || !reflect.DeepEqual(v.Parameters, req.GetParameters()) {
			return nil, status.Error(codes.AlreadyExists,
				fmt.Sprintf("Snapshot with name %s already exists", req.GetName()))
		}
		return &csi.CreateSnapshotResponse{Snapshot: &v.SnapshotCSI}, nil
	}

	// Create the snapshot and add it to the service's in-mem snapshot slice.
	snapshot := s.newSnapshot(req.GetName(), req.GetSourceVolumeId(), req.GetParameters())
	s.snapshots.Add(snapshot)

	return &csi.CreateSnapshotResponse{Snapshot: &snapshot.SnapshotCSI}, nil
}

func (s *service) DeleteSnapshot(ctx context.Context,
	req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {

	//  If the snapshot is not specified, return error
	if len(req.SnapshotId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot ID cannot be empty")
	}

	// If the snapshot does not exist then return an idempotent response.
	i, _ := s.snapshots.FindSnapshot("id", req.SnapshotId)
	if i < 0 {
		return &csi.DeleteSnapshotResponse{}, nil
	}

	// This delete logic preserves order and prevents potential memory
	// leaks. The slice's elements may not be pointers, but the structs
	// themselves have fields that are.
	s.snapshots.Delete(i)
	log.WithField("SnapshotId", req.SnapshotId).Debug("mock delete snapshot")
	return &csi.DeleteSnapshotResponse{}, nil
}

func (s *service) ListSnapshots(ctx context.Context,
	req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {

	// case 1: SnapshotId is not empty, return snapshots that match the snapshot id.
	if len(req.GetSnapshotId()) != 0 {
		return getSnapshotById(s, req)
	}

	// case 2: SourceVolumeId is not empty, return snapshots that match the source volume id.
	if len(req.GetSourceVolumeId()) != 0 {
		return getSnapshotByVolumeId(s, req)
	}

	// case 3: no parameter is set, so we return all the snapshots.
	return getAllSnapshots(s, req)
}

func getSnapshotById(s *service, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	if len(req.GetSnapshotId()) != 0 {
		i, snapshot := s.snapshots.FindSnapshot("id", req.GetSnapshotId())
		if i < 0 {
			return &csi.ListSnapshotsResponse{}, nil
		}

		if len(req.GetSourceVolumeId()) != 0 {
			if snapshot.SnapshotCSI.GetSourceVolumeId() != req.GetSourceVolumeId() {
				return &csi.ListSnapshotsResponse{}, nil
			}
		}

		return &csi.ListSnapshotsResponse{
			Entries: []*csi.ListSnapshotsResponse_Entry{
				{
					Snapshot: &snapshot.SnapshotCSI,
				},
			},
		}, nil
	}
	return nil, nil
}

func getSnapshotByVolumeId(s *service, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	if len(req.GetSourceVolumeId()) != 0 {
		i, snapshot := s.snapshots.FindSnapshot("sourceVolumeId", req.SourceVolumeId)
		if i < 0 {
			return &csi.ListSnapshotsResponse{}, nil
		}
		return &csi.ListSnapshotsResponse{
			Entries: []*csi.ListSnapshotsResponse_Entry{
				{
					Snapshot: &snapshot.SnapshotCSI,
				},
			},
		}, nil
	}
	return nil, nil
}

func getAllSnapshots(s *service, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	// Copy the mock snapshots into a new slice in order to avoid
	// locking the service's snapshot slice for the duration of the
	// ListSnapshots RPC.
	readyToUse := true
	snapshots := s.snapshots.List(readyToUse)

	var (
		ulenSnapshots = int32(len(snapshots))
		maxEntries    = req.MaxEntries
		startingToken int32
	)

	if v := req.StartingToken; v != "" {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, status.Errorf(
				codes.Aborted,
				"startingToken=%d !< int32=%d",
				startingToken, math.MaxUint32)
		}
		startingToken = int32(i)
	}

	if startingToken > ulenSnapshots {
		return nil, status.Errorf(
			codes.Aborted,
			"startingToken=%d > len(snapshots)=%d",
			startingToken, ulenSnapshots)
	}

	// Discern the number of remaining entries.
	rem := ulenSnapshots - startingToken

	// If maxEntries is 0 or greater than the number of remaining entries then
	// set maxEntries to the number of remaining entries.
	if maxEntries == 0 || maxEntries > rem {
		maxEntries = rem
	}

	var (
		i       int
		j       = startingToken
		entries = make(
			[]*csi.ListSnapshotsResponse_Entry,
			maxEntries)
	)

	for i = 0; i < len(entries); i++ {
		entries[i] = &csi.ListSnapshotsResponse_Entry{
			Snapshot: &snapshots[j],
		}
		j++
	}

	var nextToken string
	if n := startingToken + int32(i); n < ulenSnapshots {
		nextToken = fmt.Sprintf("%d", n)
	}

	return &csi.ListSnapshotsResponse{
		Entries:   entries,
		NextToken: nextToken,
	}, nil
}
