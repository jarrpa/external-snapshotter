/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connection

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/api/core/v1"
)

// CSIConnection is gRPC connection to a remote CSI driver and abstracts all
// CSI calls.
type CSIConnection interface {
	// GetDriverName returns driver name as discovered by GetPluginInfo()
	// gRPC call.
	GetDriverName(ctx context.Context) (string, error)

	// SupportsControllerCreateSnapshot returns true if the CSI driver reports
	// CREATE_DELETE_SNAPSHOT in ControllerGetCapabilities() gRPC call.
	SupportsControllerCreateSnapshot(ctx context.Context) (bool, error)

	// SupportsControllerListSnapshots returns true if the CSI driver reports
	// LIST_SNAPSHOTS in ControllerGetCapabilities() gRPC call.
	SupportsControllerListSnapshots(ctx context.Context) (bool, error)

	// CreateSnapshot creates a snapshot for a volume
	CreateSnapshot(ctx context.Context, snapshotName string, volume *v1.PersistentVolume, parameters map[string]string, snapshotterCredentials map[string]string) (driverName string, snapshotId string, timestamp int64, size int64, readyToUse bool, err error)

	// DeleteSnapshot deletes a snapshot from a volume
	DeleteSnapshot(ctx context.Context, snapshotID string, snapshotterCredentials map[string]string) (err error)

	// GetSnapshotStatus returns if a snapshot is ready to use, creation time, and restore size.
	GetSnapshotStatus(ctx context.Context, snapshotID string) (bool, int64, int64, error)

	// Probe checks that the CSI driver is ready to process requests
	Probe(ctx context.Context) error

	// Close the connection
	Close() error
}

type csiConnection struct {
	conn *grpc.ClientConn
}

var (
	_ CSIConnection = &csiConnection{}
)

func New(address string, timeout time.Duration) (CSIConnection, error) {
	conn, err := connect(address, timeout)
	if err != nil {
		return nil, err
	}
	return &csiConnection{
		conn: conn,
	}, nil
}

func connect(address string, timeout time.Duration) (*grpc.ClientConn, error) {
	glog.V(2).Infof("Connecting to %s", address)
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(time.Second),
		grpc.WithUnaryInterceptor(logGRPC),
	}
	if strings.HasPrefix(address, "/") {
		dialOptions = append(dialOptions, grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	}
	conn, err := grpc.Dial(address, dialOptions...)

	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			glog.V(4).Infof("Connection timed out")
			// subsequent GetPluginInfo will show the real connection error
			return conn, nil
		}
		if conn.GetState() == connectivity.Ready {
			glog.V(3).Infof("Connected")
			return conn, nil
		}
		glog.V(4).Infof("Still trying, connection is %s", conn.GetState())
	}
}

func (c *csiConnection) GetDriverName(ctx context.Context) (string, error) {
	client := csi.NewIdentityClient(c.conn)

	req := csi.GetPluginInfoRequest{}

	rsp, err := client.GetPluginInfo(ctx, &req)
	if err != nil {
		return "", err
	}
	name := rsp.GetName()
	if name == "" {
		return "", fmt.Errorf("name is empty")
	}
	return name, nil
}

func (c *csiConnection) Probe(ctx context.Context) error {
	client := csi.NewIdentityClient(c.conn)

	req := csi.ProbeRequest{}

	_, err := client.Probe(ctx, &req)
	if err != nil {
		return err
	}
	return nil
}

func (c *csiConnection) SupportsControllerCreateSnapshot(ctx context.Context) (bool, error) {
	client := csi.NewControllerClient(c.conn)
	req := csi.ControllerGetCapabilitiesRequest{}

	rsp, err := client.ControllerGetCapabilities(ctx, &req)
	if err != nil {
		return false, err
	}
	caps := rsp.GetCapabilities()
	for _, cap := range caps {
		if cap == nil {
			continue
		}
		rpc := cap.GetRpc()
		if rpc == nil {
			continue
		}
		if rpc.GetType() == csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT {
			return true, nil
		}
	}
	return false, nil
}

func (c *csiConnection) SupportsControllerListSnapshots(ctx context.Context) (bool, error) {
	client := csi.NewControllerClient(c.conn)
	req := csi.ControllerGetCapabilitiesRequest{}

	rsp, err := client.ControllerGetCapabilities(ctx, &req)
	if err != nil {
		return false, err
	}
	caps := rsp.GetCapabilities()
	for _, cap := range caps {
		if cap == nil {
			continue
		}
		rpc := cap.GetRpc()
		if rpc == nil {
			continue
		}
		if rpc.GetType() == csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS {
			return true, nil
		}
	}
	return false, nil
}

func (c *csiConnection) CreateSnapshot(ctx context.Context, snapshotName string, volume *v1.PersistentVolume, parameters map[string]string, snapshotterCredentials map[string]string) (string, string, int64, int64, bool, error) {
	glog.V(5).Infof("CSI CreateSnapshot: %s", snapshotName)
	if volume.Spec.CSI == nil {
		return "", "", 0, 0, false, fmt.Errorf("CSIPersistentVolumeSource not defined in spec")
	}

	client := csi.NewControllerClient(c.conn)

	driverName, err := c.GetDriverName(ctx)
	if err != nil {
		return "", "", 0, 0, false, err
	}

	req := csi.CreateSnapshotRequest{
		SourceVolumeId: volume.Spec.CSI.VolumeHandle,
		Name:           snapshotName,
		Parameters:     parameters,
		Secrets:        snapshotterCredentials,
	}

	rsp, err := client.CreateSnapshot(ctx, &req)
	if err != nil {
		return "", "", 0, 0, false, err
	}

	glog.V(5).Infof("CSI CreateSnapshot: %s driver name [%s] snapshot ID [%s] time stamp [%d] size [%d] readyToUse [%v]", snapshotName, driverName, rsp.Snapshot.SnapshotId, rsp.Snapshot.CreationTime, rsp.Snapshot.SizeBytes, rsp.Snapshot.ReadyToUse)
	creationTime, err := timestampToUnixTime(rsp.Snapshot.CreationTime)
	if err != nil {
		return "", "", 0, 0, false, err
	}
	return driverName, rsp.Snapshot.SnapshotId, creationTime, rsp.Snapshot.SizeBytes, rsp.Snapshot.ReadyToUse, nil
}

func (c *csiConnection) DeleteSnapshot(ctx context.Context, snapshotID string, snapshotterCredentials map[string]string) (err error) {
	client := csi.NewControllerClient(c.conn)

	req := csi.DeleteSnapshotRequest{
		SnapshotId: snapshotID,
		Secrets:    snapshotterCredentials,
	}

	if _, err := client.DeleteSnapshot(ctx, &req); err != nil {
		return err
	}

	return nil
}

func (c *csiConnection) GetSnapshotStatus(ctx context.Context, snapshotID string) (bool, int64, int64, error) {
	client := csi.NewControllerClient(c.conn)

	req := csi.ListSnapshotsRequest{
		SnapshotId: snapshotID,
	}

	rsp, err := client.ListSnapshots(ctx, &req)
	if err != nil {
		return false, 0, 0, err
	}

	if rsp.Entries == nil || len(rsp.Entries) == 0 {
		return false, 0, 0, fmt.Errorf("can not find snapshot for snapshotID %s", snapshotID)
	}

	creationTime, err := timestampToUnixTime(rsp.Entries[0].Snapshot.CreationTime)
	if err != nil {
		return false, 0, 0, err
	}
	return rsp.Entries[0].Snapshot.ReadyToUse, creationTime, rsp.Entries[0].Snapshot.SizeBytes, nil
}

func (c *csiConnection) Close() error {
	return c.conn.Close()
}

func logGRPC(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	glog.V(5).Infof("GRPC call: %s", method)
	glog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	err := invoker(ctx, method, req, reply, cc, opts...)
	glog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(reply))
	glog.V(5).Infof("GRPC error: %v", err)
	return err
}

func timestampToUnixTime(t *timestamp.Timestamp) (int64, error) {
	time, err := ptypes.Timestamp(t)
	if err != nil {
		return -1, err
	}
	// TODO: clean this up, we probably don't need this translation layer
	// and can just use time.Time
	return time.UnixNano(), nil
}
