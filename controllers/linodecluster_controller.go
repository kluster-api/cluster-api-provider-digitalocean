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
	"time"

	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-linode/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-linode/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-linode/cloud/services/networking"
	dnsutil "sigs.k8s.io/cluster-api-provider-linode/util/dns"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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

// LinodeClusterReconciler reconciles a LinodeCluster object.
type LinodeClusterReconciler struct {
	client.Client
	Recorder record.EventRecorder
}

func (r *LinodeClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.LinodeCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	// Add a watch on clusterv1.Cluster object for unpause notifications.
	if err = c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("LinodeCluster"))),
		predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=linodeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=linodeclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *LinodeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	linodecluster := &infrav1.LinodeCluster{}
	if err := r.Get(ctx, req.NamespacedName, linodecluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, linodecluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:    r.Client,
		Logger:    log,
		Cluster:   cluster,
		LinodeCluster: linodecluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted clusters
	if !linodecluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	return r.reconcile(ctx, clusterScope)
}

func (r *LinodeClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling LinodeCluster")
	linodecluster := clusterScope.LinodeCluster
	// If the LinodeCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(linodecluster, infrav1.ClusterFinalizer)

	networkingsvc := networking.NewService(ctx, clusterScope)
	apiServerLoadbalancer := clusterScope.APIServerLoadbalancers()
	apiServerLoadbalancer.ApplyDefault()

	apiServerLoadbalancerRef := clusterScope.APIServerLoadbalancersRef()
	loadbalancer, err := networkingsvc.GetLoadBalancer(apiServerLoadbalancerRef.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		loadbalancer, err = networkingsvc.CreateLoadBalancer(apiServerLoadbalancer)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to create load balancers for LinodeCluster %s/%s", linodecluster.Namespace, linodecluster.Name)
		}

		r.Recorder.Eventf(linodecluster, corev1.EventTypeNormal, "LoadBalancerCreated", "Created new load balancers - %s", loadbalancer.Name)
	}

	apiServerLoadbalancerRef.ResourceID = loadbalancer.ID
	apiServerLoadbalancerRef.ResourceStatus = infrav1.LinodeResourceStatus(loadbalancer.Status)

	if apiServerLoadbalancerRef.ResourceStatus != infrav1.LinodeResourceStatusRunning && loadbalancer.IP == "" {
		clusterScope.Info("Waiting on API server Global IP Address")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	r.Recorder.Eventf(linodecluster, corev1.EventTypeNormal, "LoadBalancerReady", "LoadBalancer got an IP Address - %s", loadbalancer.IP)

	var controlPlaneEndpoint = loadbalancer.IP
	if linodecluster.Spec.ControlPlaneDNS != nil {
		clusterScope.Info("Verifying LB DNS Record")
		// ensure DNS record is created and use it as control plane endpoint
		recordSpec := linodecluster.Spec.ControlPlaneDNS
		controlPlaneEndpoint = fmt.Sprintf("%s.%s", recordSpec.Name, recordSpec.Domain)
		dRecord, err := networkingsvc.GetDomainRecord(
			recordSpec.Domain,
			recordSpec.Name,
			"A",
		)

		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed verify DNS record for LB Name %s.%s",
				recordSpec.Name, recordSpec.Domain)
		}

		if dRecord == nil || dRecord.Data != loadbalancer.IP {
			clusterScope.Info("Ensuring LB DNS Record is in place")
			clusterScope.SetControlPlaneDNSRecordReady(false)
			if err := networkingsvc.UpsertDomainRecord(
				recordSpec.Domain,
				recordSpec.Name,
				"A",
				loadbalancer.IP,
			); err != nil {
				return reconcile.Result{}, errors.Wrap(err, "failed to reconcile LB DNS record")
			}
		}

		// If the record has never been ready we need to check whether it has
		// been propagated or not. Updating the record in the DNS API does not
		// mean it is already advertised at the DNS server. If the DNS is slower
		// than our reconciliation is, we'd fall into a case where our
		// reconciler hits an NXDOMAIN which is then stored in the negative
		// cache, so all our retries would fail until the cache TTL is up. This
		// propagation check works around the DNS cache problem by directly
		// making DNS queries and not going through system resolvers.
		if !clusterScope.LinodeCluster.Status.ControlPlaneDNSRecordReady {
			propagated, err := dnsutil.CheckDNSPropagated(dnsutil.ToFQDN(recordSpec.Name, recordSpec.Domain), loadbalancer.IP)
			if err != nil {
				return reconcile.Result{}, errors.Wrap(err, "failed to check DNS propagation")
			}

			if !propagated {
				clusterScope.Info("Waiting for DNS record to be propagated")
				return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}

			clusterScope.Info("DNS record is propagated - set LinodeCluster ControlPlaneDNSRecordReady status to ready")
			clusterScope.SetControlPlaneDNSRecordReady(true)
		}

		clusterScope.Info("LB DNS Record is already ready")
		r.Recorder.Eventf(linodecluster, corev1.EventTypeNormal, "DomainRecordReady", "DNS Record '%s.%s' with IP '%s'", recordSpec.Name, recordSpec.Domain, loadbalancer.IP)
	}

	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: int32(apiServerLoadbalancer.Port),
	})

	clusterScope.Info("Set LinodeCluster status to ready")
	clusterScope.SetReady()
	r.Recorder.Eventf(linodecluster, corev1.EventTypeNormal, "LinodeClusterReady", "LinodeCluster %s - has ready status", clusterScope.Name())
	return reconcile.Result{}, nil
}

func (r *LinodeClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling delete LinodeCluster")
	linodecluster := clusterScope.LinodeCluster
	networkingsvc := networking.NewService(ctx, clusterScope)
	apiServerLoadbalancerRef := clusterScope.APIServerLoadbalancersRef()

	if linodecluster.Spec.ControlPlaneDNS != nil {
		recordSpec := linodecluster.Spec.ControlPlaneDNS
		if err := networkingsvc.DeleteDomainRecord(recordSpec.Domain, recordSpec.Name, "A"); err != nil {
			return reconcile.Result{}, err
		}
	}

	loadbalancer, err := networkingsvc.GetLoadBalancer(apiServerLoadbalancerRef.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}

	if loadbalancer == nil {
		clusterScope.V(2).Info("Unable to locate load balancer")
		r.Recorder.Eventf(linodecluster, corev1.EventTypeWarning, "NoLoadBalancerFound", "Unable to find matching load balancer")
		controllerutil.RemoveFinalizer(linodecluster, infrav1.ClusterFinalizer)
		return reconcile.Result{}, nil
	}

	if err := networkingsvc.DeleteLoadBalancer(loadbalancer.ID); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "error deleting load balancer for LinodeCluster %s/%s", linodecluster.Namespace, linodecluster.Name)
	}

	r.Recorder.Eventf(linodecluster, corev1.EventTypeNormal, "LoadBalancerDeleted", "Deleted an LoadBalancer - %s", loadbalancer.Name)
	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(linodecluster, infrav1.ClusterFinalizer)
	return reconcile.Result{}, nil
}
