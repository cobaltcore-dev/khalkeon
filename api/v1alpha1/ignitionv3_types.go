// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IgnitionV3Spec defines the desired state of IgnitionV3.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.targetSecret) || has(self.targetSecret)", message="targetSecret is required once set"
type IgnitionV3Spec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="targetSecret is immutable"
	TargetSecret *v1.LocalObjectReference `json:"targetSecret,omitempty"`

	Config `json:",inline"`
}

// IgnitionV3Status defines the observed state of IgnitionV3.
type IgnitionV3Status struct {
	// Conditions represents the latest available observations of the ignition's current state.
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// TargetIgnitions is a list of Ignitions with TargetSecret that merged this ignition
	TargetIgnitions []v1.LocalObjectReference `json:"targetIgnitions,omitempty"`
	// TODO what if merge is changed and Ignition is no longer used for a secret. It will trigger unnecessary reconciliation.
}

const (
	ConfigurationType = "Configuration"
	SecretType        = "Secret"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ign

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
