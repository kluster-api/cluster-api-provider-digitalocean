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

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-linode/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-linode/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-linode/cloud/services/computes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// LinodeMachineReconciler reconciles a LinodeMachine object.
type LinodeMachineReconciler struct {
	client.Client
	Recorder record.EventRecorder
}

func (r *LinodeMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.LinodeMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("LinodeMachine"))),
		).
		Watches(
			&source.Kind{Type: &infrav1.LinodeCluster{}},
			handler.EnqueueRequestsFromMapFunc(r.LinodeClusterToLinodeMachines(ctx)),
		).
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &infrav1.LinodeMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to LinodeMachines")
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

func (r *LinodeMachineReconciler) LinodeClusterToLinodeMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*infrav1.LinodeCluster)
		if !ok {
			log.Error(errors.Errorf("expected a LinodeCluster but got a %T", o), "failed to get LinodeMachine for LinodeCluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines")
			return nil
		}
		for _, m := range machineList.Items {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=linodemachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=linodemachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *LinodeMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	linodemachine := &infrav1.LinodeMachine{}
	if err := r.Get(ctx, req.NamespacedName, linodemachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, linodemachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	linodecluster := &infrav1.LinodeCluster{}
	linodeclusterNamespacedName := client.ObjectKey{
		Namespace: linodemachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, linodeclusterNamespacedName, linodecluster); err != nil {
		log.Info("DOluster is not available yet")
		return reconcile.Result{}, nil
	}

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:    r.Client,
		Logger:    log,
		Cluster:   cluster,
		LinodeCluster: linodecluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:    log,
		Client:    r.Client,
		Cluster:   cluster,
		Machine:   machine,
		LinodeCluster: linodecluster,
		LinodeMachine: linodemachine,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any LinodeMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !linodemachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcile(ctx, machineScope, clusterScope)
}

func (r *LinodeMachineReconciler) reconcileVolumes(ctx context.Context, mscope *scope.MachineScope, cscope *scope.ClusterScope) (reconcile.Result, error) {
	mscope.Info("Reconciling LinodeMachine Volumes")
	computesvc := computes.NewService(ctx, cscope)
	linodemachine := mscope.LinodeMachine
	for _, disk := range linodemachine.Spec.DataDisks {
		volName := infrav1.DataDiskName(linodemachine, disk.NameSuffix)
		vol, err := computesvc.GetVolumeByName(volName)
		if err != nil {
			return reconcile.Result{}, err
		}
		if vol == nil {
			_, err = computesvc.CreateVolume(disk, volName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		// TODO(gottwald): reconcile disk resizes here (at least grow)
	}
	return reconcile.Result{}, nil
}

func (r *LinodeMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling LinodeMachine")
	linodemachine := machineScope.LinodeMachine
	// If the LinodeMachine is in an error state, return early.
	if linodemachine.Status.FailureReason != nil || linodemachine.Status.FailureMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the LinodeMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(linodemachine, infrav1.MachineFinalizer)

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	// Make sure the droplet volumes are reconciled
	if result, err := r.reconcileVolumes(ctx, machineScope, clusterScope); err != nil {
		return result, fmt.Errorf("failed to reconcile volumes: %w", err)
	}

	computesvc := computes.NewService(ctx, clusterScope)
	droplet, err := computesvc.GetDroplet(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}
	if droplet == nil {
		droplet, err = computesvc.CreateDroplet(machineScope)
		if err != nil {
			err = errors.Errorf("Failed to create droplet instance for LinodeMachine %s/%s: %v", linodemachine.Namespace, linodemachine.Name, err)
			r.Recorder.Event(linodemachine, corev1.EventTypeWarning, "InstanceCreatingError", err.Error())
			machineScope.SetInstanceStatus(infrav1.LinodeResourceStatusErrored)
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(linodemachine, corev1.EventTypeNormal, "InstanceCreated", "Created new droplet instance - %s", droplet.Name)
	}

	machineScope.SetProviderID(strconv.Itoa(droplet.ID))
	machineScope.SetInstanceStatus(infrav1.LinodeResourceStatus(droplet.Status))

	addrs, err := computesvc.GetDropletAddress(droplet)
	if err != nil {
		machineScope.SetFailureMessage(errors.New("failed to getting droplet address"))
		return reconcile.Result{}, err
	}
	machineScope.SetAddresses(addrs)

	// Proceed to reconcile the LinodeMachine state.
	switch infrav1.LinodeResourceStatus(droplet.Status) {
	case infrav1.LinodeResourceStatusNew:
		machineScope.Info("Machine instance is pending", "instance-id", machineScope.GetInstanceID())
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	case infrav1.LinodeResourceStatusRunning:
		machineScope.Info("Machine instance is active", "instance-id", machineScope.GetInstanceID())
		machineScope.SetReady()
		r.Recorder.Eventf(linodemachine, corev1.EventTypeNormal, "LinodeMachineReady", "LinodeMachine %s - has ready status", droplet.Name)
		return reconcile.Result{}, nil
	default:
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("Instance status %q is unexpected", droplet.Status))
		return reconcile.Result{}, nil
	}
}
func (r *LinodeMachineReconciler) reconcileDeleteVolumes(ctx context.Context, mscope *scope.MachineScope, cscope *scope.ClusterScope) (reconcile.Result, error) {
	mscope.Info("Reconciling delete LinodeMachine Volumes")
	computesvc := computes.NewService(ctx, cscope)
	linodemachine := mscope.LinodeMachine
	for _, disk := range linodemachine.Spec.DataDisks {
		volName := infrav1.DataDiskName(linodemachine, disk.NameSuffix)
		vol, err := computesvc.GetVolumeByName(volName)
		if err != nil {
			return reconcile.Result{}, err
		}
		if vol == nil {
			continue
		}
		if err = computesvc.DeleteVolume(vol.ID); err != nil {
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(linodemachine, corev1.EventTypeNormal, "VolumeDeleted", "Deleted the storage volume - %s", vol.Name)
	}
	return reconcile.Result{}, nil
}

func (r *LinodeMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling delete LinodeMachine")
	linodemachine := machineScope.LinodeMachine

	computesvc := computes.NewService(ctx, clusterScope)
	droplet, err := computesvc.GetDroplet(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}

	if droplet != nil {
		if err := computesvc.DeleteDroplet(machineScope.GetInstanceID()); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		clusterScope.V(2).Info("Unable to locate droplet instance")
		r.Recorder.Eventf(linodemachine, corev1.EventTypeWarning, "NoInstanceFound", "Skip deleting")
	}
	if result, err := r.reconcileDeleteVolumes(ctx, machineScope, clusterScope); err != nil {
		return result, fmt.Errorf("failed to reconcile delete volumes: %w", err)
	}
	r.Recorder.Eventf(linodemachine, corev1.EventTypeNormal, "InstanceDeleted", "Deleted a instance - %s", machineScope.Name())
	controllerutil.RemoveFinalizer(linodemachine, infrav1.MachineFinalizer)
	return reconcile.Result{}, nil
}
