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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-linode/api/v1alpha4"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	LinodeClients
	Client    client.Client
	Logger    logr.Logger
	Cluster   *clusterv1.Cluster
	Machine   *clusterv1.Machine
	LinodeCluster *infrav1.LinodeCluster
	LinodeMachine *infrav1.LinodeMachine
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration
// both LinodeClusterReconciler and LinodeMachineReconciler.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.LinodeCluster == nil {
		return nil, errors.New("LinodeCluster is required when creating a MachineScope")
	}
	if params.LinodeMachine == nil {
		return nil, errors.New("LinodeMachine is required when creating a MachineScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.LinodeMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		client:      params.Client,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		LinodeCluster:   params.LinodeCluster,
		LinodeMachine:   params.LinodeMachine,
		Logger:      params.Logger,
		patchHelper: helper,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster   *clusterv1.Cluster
	Machine   *clusterv1.Machine
	LinodeCluster *infrav1.LinodeCluster
	LinodeMachine *infrav1.LinodeMachine
}

// Close the MachineScope by updating the machine spec, machine status.
func (m *MachineScope) Close() error {
	return m.patchHelper.Patch(context.TODO(), m.LinodeMachine)
}

// Name returns the LinodeMachine name.
func (m *MachineScope) Name() string {
	return m.LinodeMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.LinodeMachine.Namespace
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return infrav1.APIServerRoleTagValue
	}
	return infrav1.NodeRoleTagValue
}

// GetProviderID returns the LinodeMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.LinodeMachine.Spec.ProviderID != nil {
		return *m.LinodeMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the LinodeMachine providerID in spec from droplet id.
func (m *MachineScope) SetProviderID(dropletID string) {
	pid := fmt.Sprintf("linode://%s", dropletID)
	m.LinodeMachine.Spec.ProviderID = pointer.StringPtr(pid)
}

// GetInstanceID returns the LinodeMachine droplet instance id by parsing Spec.ProviderID.
func (m *MachineScope) GetInstanceID() string {
	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return ""
	}
	return parsed.ID()
}

// GetInstanceStatus returns the LinodeMachine droplet instance status from the status.
func (m *MachineScope) GetInstanceStatus() *infrav1.LinodeResourceStatus {
	return m.LinodeMachine.Status.InstanceStatus
}

// SetInstanceStatus sets the LinodeMachine droplet id.
func (m *MachineScope) SetInstanceStatus(v infrav1.LinodeResourceStatus) {
	m.LinodeMachine.Status.InstanceStatus = &v
}

// SetReady sets the LinodeMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.LinodeMachine.Status.Ready = true
}

// SetFailureMessage sets the LinodeMachine status error message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.LinodeMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

// SetFailureReason sets the LinodeMachine status error reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.LinodeMachine.Status.FailureReason = &v
}

// SetAddresses sets the address status.
func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.LinodeMachine.Status.Addresses = addrs
}

// AdditionalTags returns AdditionalTags from the scope's LinodeMachine. The returned value will never be nil.
func (m *MachineScope) AdditionalTags() infrav1.Tags {
	if m.LinodeMachine.Spec.AdditionalTags == nil {
		m.LinodeMachine.Spec.AdditionalTags = infrav1.Tags{}
	}

	return m.LinodeMachine.Spec.AdditionalTags.DeepCopy()
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for LinodeMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return string(value), nil
}
