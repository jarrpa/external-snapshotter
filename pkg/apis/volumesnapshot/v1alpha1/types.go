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

package v1alpha1

import (
	core_v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VolumeSnapshotContentResourcePlural is "volumesnapshotcontents"
	VolumeSnapshotContentResourcePlural = "volumesnapshotcontents"
	// VolumeSnapshotResourcePlural is "volumesnapshots"
	VolumeSnapshotResourcePlural = "volumesnapshots"
	// VolumeSnapshotClassResourcePlural is "volumesnapshotclasses"
	VolumeSnapshotClassResourcePlural = "volumesnapshotclasses"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshot is a user's request for taking a snapshot. Upon successful creation of the actual
// snapshot by the volume provider it is bound to the corresponding VolumeSnapshotContent.
// Only the VolumeSnapshot object is accessible to the user in the namespace.
type VolumeSnapshot struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired characteristics of a snapshot requested by a user.
	Spec VolumeSnapshotSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status represents the latest observed state of the snapshot
	// +optional
	Status VolumeSnapshotStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotList is a list of VolumeSnapshot objects
type VolumeSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of VolumeSnapshots
	Items []VolumeSnapshot `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// VolumeSnapshotSpec describes the common attributes of a volume snapshot
type VolumeSnapshotSpec struct {
	// Source has the information about where the snapshot is created from.
	// In Alpha version, only PersistentVolumeClaim is supported as the source.
	// If not specified, user can create VolumeSnapshotContent and bind it with VolumeSnapshot manually.
	// +optional
	Source *TypedLocalObjectReference `json:"source" protobuf:"bytes,1,opt,name=source"`

	// SnapshotContentName binds the VolumeSnapshot object with the VolumeSnapshotContent
	// +optional
	SnapshotContentName string `json:"snapshotContentName" protobuf:"bytes,2,opt,name=snapshotContentName"`

	// Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default snapshot class will
	// be used if it is available.
	// +optional
	VolumeSnapshotClassName string `json:"snapshotClassName" protobuf:"bytes,3,opt,name=snapshotClassName"`

	// The complete size of the volume snapshot
	// +optional
	Size int64 `json:"size,omitempty" protobuf:"varint,4,opt,name=size"`
}

// VolumeSnapshotStatus is the status of the VolumeSnapshot
type VolumeSnapshotStatus struct {
	// CreationTime is the time the snapshot was successfully created. If it is set,
	// it means the snapshot was created; Otherwise the snapshot was not created.
	// +optional
	CreationTime *metav1.Time `json:"createdAt" protobuf:"bytes,1,opt,name=createdAt"`

	// Ready is set to true only if the snapshot is ready to use (e.g., finish uploading if
	// there is an uploading phase) and also VolumeSnapshot and its VolumeSnapshotContent
	// bind correctly with each other. If any of the above condition is not true, Ready is
	// set to false
	// +optional
	Ready bool `json:"ready" protobuf:"varint,2,opt,name=ready"`

	// The last error encountered during create snapshot operation, if any.
	// This field must only be set by the entity completing the create snapshot
	// operation, i.e. the external-snapshotter.
	// +optional
	Error *storage.VolumeError
}

// TypedLocalObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.
// TODO: After TypedLocalObjectReference is merged into the in-tree core API, this will be replaced.
type TypedLocalObjectReference struct {
	// Name of the referent.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Kind of the referent.
	// +optional
	Kind string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotClass describes the parameters used by storage system when
// provisioning VolumeSnapshots from PVCs.
// The name of a VolumeSnapshotClass object is significant, and is how users can request a particular class.
type VolumeSnapshotClass struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Snapshotter is the driver expected to handle this VolumeSnapshotClass.
	Snapshotter string `json:"snapshotter" protobuf:"bytes,2,opt,name=snapshotter"`

	// Parameters holds parameters for the snapshotter.
	// These values are opaque to the system and are passed directly
	// to the snapshotter.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotClassList is a collection of snapshot classes.
type VolumeSnapshotClassList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of VolumeSnapshotClasses
	Items []VolumeSnapshotClass `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotContent represents the actual "on-disk" snapshot object
type VolumeSnapshotContent struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec represents the desired state of the snapshot data
	Spec VolumeSnapshotContentSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotContentList is a list of VolumeSnapshotContent objects
type VolumeSnapshotContentList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of VolumeSnapshotContents
	Items []VolumeSnapshotContent `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// VolumeSnapshotContentSpec is the spec of the volume snapshot data
type VolumeSnapshotContentSpec struct {
	// Source represents the location and type of the volume snapshot
	VolumeSnapshotSource `json:",inline" protobuf:"bytes,1,opt,name=volumeSnapshotSource"`

	// VolumeSnapshotRef is part of bi-directional binding between VolumeSnapshot
	// and VolumeSnapshotContent. It becomes non-nil when bound.
	// +optional
	VolumeSnapshotRef *core_v1.ObjectReference `json:"volumeSnapshotRef" protobuf:"bytes,2,opt,name=volumeSnapshotRef"`

	// PersistentVolumeRef represents the PersistentVolume that the snapshot has been
	// taken from. It becomes non-nil when VolumeSnapshot and VolumeSnapshotContent are bound.
	// +optional
	PersistentVolumeRef *core_v1.ObjectReference `json:"persistentVolumeRef" protobuf:"bytes,3,opt,name=persistentVolumeRef"`

	// Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default snapshot class will
	// be used if it is available.
	// +optional
	VolumeSnapshotClassName string `json:"snapshotClassName" protobuf:"bytes,4,opt,name=snapshotClassName"`
}

// VolumeSnapshotSource represents the actual location and type of the snapshot. Only one of its members may be specified.
type VolumeSnapshotSource struct {
	// CSI (Container Storage Interface) represents storage that handled by an external CSI Volume Driver (Alpha feature).
	// +optional
	CSI *CSIVolumeSnapshotSource `json:"csiVolumeSnapshotSource,omitempty"`
}

// Represents the source from CSI volume snapshot
type CSIVolumeSnapshotSource struct {
	// Driver is the name of the driver to use for this snapshot.
	// This MUST be the same name returned by the CSI GetPluginName() call for
	// that driver.
	// Required.
	Driver string `json:"driver" protobuf:"bytes,1,opt,name=driver"`

	// SnapshotHandle is the unique snapshot id returned by the CSI volume
	// plugin’s CreateSnapshot to refer to the snapshot on all subsequent calls.
	// Required.
	SnapshotHandle string `json:"snapshotHandle" protobuf:"bytes,2,opt,name=snapshotHandle"`

	// Timestamp when the point-in-time snapshot is taken on the storage
	// system. This timestamp will be generated by the CSI volume driver after
	// the snapshot is cut. The format of this field should be a Unix nanoseconds
	// time encoded as an int64. On Unix, the command `date +%s%N` returns
	// the  current time in nanoseconds since 1970-01-01 00:00:00 UTC.
	// This field is required in the CSI spec but optional here to support static binding.
	// +optional
	CreatedAt int64 `json:"createdAt,omitempty" protobuf:"varint,3,opt,name=createdAt"`

	// The complete size of the volume snapshot
	// +optional
	Size int64 `json:"size,omitempty" protobuf:"varint,4,opt,name=size"`
}