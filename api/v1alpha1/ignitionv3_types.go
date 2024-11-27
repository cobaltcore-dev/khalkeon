/*
Copyright 2024.

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
	ignitiontypes "github.com/coreos/ignition/v2/config/v3_5/types"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IgnitionV3Spec defines the desired state of IgnitionV3.
type IgnitionV3Spec struct {
	TargetSecret v1.LocalObjectReference `json:"targetSecret,omitempty"`

	// Copied from ignitiontypes.Config
	Ignition        Ignition                      `json:"ignition"`
	KernelArguments ignitiontypes.KernelArguments `json:"kernelArguments,omitempty"`
	Passwd          ignitiontypes.Passwd          `json:"passwd,omitempty"`
	Storage         ignitiontypes.Storage         `json:"storage,omitempty"`
	Systemd         ignitiontypes.Systemd         `json:"systemd,omitempty"`
}

// Copied from ignitiontypes.Ignition
type Ignition struct {
	Config   IgnitionConfig         `json:"config,omitempty"`
	Proxy    ignitiontypes.Proxy    `json:"proxy,omitempty"`
	Security ignitiontypes.Security `json:"security,omitempty"`
	Timeouts ignitiontypes.Timeouts `json:"timeouts,omitempty"`
	Version  string                 `json:"version"`
}

// Copied from ignitiontypes.IgnitionConfig
type IgnitionConfig struct {
	Merge   metav1.LabelSelector    `json:"merge,omitempty"`
	Replace v1.LocalObjectReference `json:"replace,omitempty"`
}

// COMMENT this part is needed as some of subfileds of Storage don't have json tags and CRD yaml can't be generated.
// However there is still a problem with DeepCopy.
// // Copied from ignitiontypes.Storage
// type Storage struct {
// 	Directories []Directory                `json:"directories,omitempty"`
// 	Disks       []ignitiontypes.Disk       `json:"disks,omitempty"`
// 	Files       []File                     `json:"files,omitempty"`
// 	Filesystems []ignitiontypes.Filesystem `json:"filesystems,omitempty"`
// 	Links       []Link                     `json:"links,omitempty"`
// 	Luks        []ignitiontypes.Luks       `json:"luks,omitempty"`
// 	Raid        []ignitiontypes.Raid       `json:"raid,omitempty"`
// }

// // Copied from ignitiontypes.Directory
// type Directory struct {
// 	ignitiontypes.Node               `json:"node,omitempty"`
// 	ignitiontypes.DirectoryEmbedded1 `json:"directoryEmbedded1,omitempty"`
// }

// // Copied from ignitiontypes.File
// type File struct {
// 	ignitiontypes.Node          `json:"node,omitempty"`
// 	ignitiontypes.FileEmbedded1 `json:"fileEmbedded1,omitempty"`
// }

// // Copied from ignitiontypes.Link
// type Link struct {
// 	ignitiontypes.Node          `json:"node,omitempty"`
// 	ignitiontypes.LinkEmbedded1 `json:"linkEmbedded1,omitempty"`
// }

// IgnitionV3Status defines the observed state of IgnitionV3.
type IgnitionV3Status struct {
	// Conditions represents the latest available observations of the ignition's current state.
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// IgnitionV3 is the Schema for the ignitionv3s API.
type IgnitionV3 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IgnitionV3Spec   `json:"spec,omitempty"`
	Status IgnitionV3Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IgnitionV3List contains a list of IgnitionV3.
type IgnitionV3List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IgnitionV3 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IgnitionV3{}, &IgnitionV3List{})
}
