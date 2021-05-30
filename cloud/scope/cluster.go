/*
Copyright 2020 The Kubernetes Authors.

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

package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-linode/api/v1alpha4"

	"k8s.io/klog/v2/klogr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	LinodeClients
	Client    client.Client
	Logger    logr.Logger
	Cluster   *clusterv1.Cluster
	LinodeCluster *infrav1.LinodeCluster
}

// NewClusterScope creates a new ClusterScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on LinodeClusterReconciler.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}
	if params.LinodeCluster == nil {
		return nil, errors.New("LinodeCluster is required when creating a ClusterScope")
	}
	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	session, err := params.LinodeClients.Session()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DO session")
	}

	if params.LinodeClients.Actions == nil {
		params.LinodeClients.Actions = session.Actions
	}

	if params.LinodeClients.Droplets == nil {
		params.LinodeClients.Droplets = session.Droplets
	}

	if params.LinodeClients.Storage == nil {
		params.LinodeClients.Storage = session.Storage
	}

	if params.LinodeClients.Images == nil {
		params.LinodeClients.Images = session.Images
	}

	if params.LinodeClients.Keys == nil {
		params.LinodeClients.Keys = session.Keys
	}

	if params.LinodeClients.LoadBalancers == nil {
		params.LinodeClients.LoadBalancers = session.LoadBalancers
	}

	if params.LinodeClients.Domains == nil {
		params.LinodeClients.Domains = session.Domains
	}

	helper, err := patch.NewHelper(params.LinodeCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &ClusterScope{
		Logger:      params.Logger,
		client:      params.Client,
		LinodeClients:   params.LinodeClients,
		Cluster:     params.Cluster,
		LinodeCluster:   params.LinodeCluster,
		patchHelper: helper,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	LinodeClients
	Cluster   *clusterv1.Cluster
	LinodeCluster *infrav1.LinodeCluster
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.LinodeCluster)
}

// Name returns the cluster name.
func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

// Namespace returns the cluster namespace.
func (s *ClusterScope) Namespace() string {
	return s.Cluster.GetNamespace()
}

func (s *ClusterScope) UID() string {
	return string(s.Cluster.UID)
}

// Region returns the cluster region.
func (s *ClusterScope) Region() string {
	return s.LinodeCluster.Spec.Region
}

// Network returns the cluster network object.
func (s *ClusterScope) Network() *infrav1.LinodeNetworkResource {
	return &s.LinodeCluster.Status.Network
}

// SetReady sets the LinodeCluster Ready Status.
func (s *ClusterScope) SetReady() {
	s.LinodeCluster.Status.Ready = true
}

// SetControlPlaneDNSRecordReady sets the LinodeCluster ControlPlaneDNSRecordReady Status.
func (s *ClusterScope) SetControlPlaneDNSRecordReady(ready bool) {
	s.LinodeCluster.Status.ControlPlaneDNSRecordReady = ready
}

// SetControlPlaneEndpoint sets the LinodeCluster status APIEndpoints.
func (s *ClusterScope) SetControlPlaneEndpoint(apiEndpoint clusterv1.APIEndpoint) {
	s.LinodeCluster.Spec.ControlPlaneEndpoint = apiEndpoint
}

// APIServerLoadbalancers get the LinodeCluster Spec Network APIServerLoadbalancers.
func (s *ClusterScope) APIServerLoadbalancers() *infrav1.LinodeLoadBalancer {
	return &s.LinodeCluster.Spec.Network.APIServerLoadbalancers
}

// APIServerLoadbalancersRef get the LinodeCluster status Network APIServerLoadbalancersRef.
func (s *ClusterScope) APIServerLoadbalancersRef() *infrav1.LinodeResourceReference {
	return &s.LinodeCluster.Status.Network.APIServerLoadbalancersRef
}

// VPC gets the LinodeCluster Spec Network VPC.
func (s *ClusterScope) VPC() *infrav1.LinodeVPC {
	return &s.LinodeCluster.Spec.Network.VPC
}
