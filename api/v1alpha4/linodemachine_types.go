/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha4

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileLinodeMachine to clean up Linode resources associated with LinodeMachine before
	// removing it from the apiserver.
	MachineFinalizer = "linodemachine.infrastructure.cluster.x-k8s.io"
)

// LinodeMachineSpec defines the desired state of LinodeMachine.
type LinodeMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`
	// Droplet size. It must be known Linode droplet size. See https://developers.linode.com/documentation/v2/#list-all-sizes
	Size string `json:"size"`
	// Droplet image can be image id or slug. See https://developers.linode.com/documentation/v2/#list-all-images
	Image intstr.IntOrString `json:"image"`
	// DataDisks specifies the parameters that are used to add one or more data disks to the machine
	DataDisks []DataDisk `json:"dataDisks,omitempty"`
	// SSHKeys is the ssh key id or fingerprint to attach in Linode droplet.
	// It must be available on Linode account. See https://developers.linode.com/documentation/v2/#list-all-keys
	SSHKeys []intstr.IntOrString `json:"sshKeys"`
	// AdditionalTags is an optional set of tags to add to Linode resources managed by the Linode provider.
	// +optional
	AdditionalTags Tags `json:"additionalTags,omitempty"`
}

// LinodeMachineStatus defines the observed state of LinodeMachine.
type LinodeMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the Linode droplet associated addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceStatus is the status of the Linode droplet instance for this machine.
	// +optional
	InstanceStatus *LinodeResourceStatus `json:"instanceStatus,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=linodemachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this LinodeMachine belongs"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.instanceStatus",description="Linode droplet instance state"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
// +kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="Linode droplet instance ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this LinodeMachine"

// LinodeMachine is the Schema for the linodemachines API.
type LinodeMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinodeMachineSpec   `json:"spec,omitempty"`
	Status LinodeMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LinodeMachineList contains a list of LinodeMachine.
type LinodeMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LinodeMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LinodeMachine{}, &LinodeMachineList{})
}
