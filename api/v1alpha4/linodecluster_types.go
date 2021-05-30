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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

const (
	// ClusterFinalizer allows ReconcileLinodeCluster to clean up Linode resources associated with LinodeCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "linodecluster.infrastructure.cluster.x-k8s.io"
)

// LinodeClusterSpec defines the desired state of LinodeCluster.
type LinodeClusterSpec struct {
	// The Linode Region the cluster lives in. It must be one of available
	// region on Linode. See
	// https://developers.linode.com/documentation/v2/#list-all-regions
	Region string `json:"region"`
	// Network configurations
	// +optional
	Network LinodeNetwork `json:"network,omitempty"`
	// ControlPlaneEndpoint represents the endpoint used to communicate with the
	// control plane. If ControlPlaneDNS is unset, the DO load-balancer IP
	// of the Kubernetes API Server is used.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
	// ControlPlaneDNS is a managed DNS name that points to the load-balancer
	// IP used for the ControlPlaneEndpoint.
	// +optional
	ControlPlaneDNS *LinodeControlPlaneDNS `json:"controlPlaneDNS,omitempty"`
}

// LinodeClusterStatus defines the observed state of LinodeCluster.
type LinodeClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`
	// ControlPlaneDNSRecordReady denotes that the DNS record is ready and
	// propagated to the DO DNS servers.
	// +optional
	ControlPlaneDNSRecordReady bool `json:"controlPlaneDNSRecordReady,omitempty"`
	// Network encapsulates all things related to Linode network.
	// +optional
	Network LinodeNetworkResource `json:"network,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=linodeclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this LinodeCluster belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for Linode droplet instances"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.ControlPlaneEndpoint",description="API Endpoint",priority=1

// LinodeCluster is the Schema for the LinodeClusters API.
type LinodeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinodeClusterSpec   `json:"spec,omitempty"`
	Status LinodeClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LinodeClusterList contains a list of LinodeCluster.
type LinodeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LinodeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LinodeCluster{}, &LinodeClusterList{})
}
