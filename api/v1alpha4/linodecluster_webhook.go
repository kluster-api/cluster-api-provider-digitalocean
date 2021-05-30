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
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var _ = logf.Log.WithName("linodecluster-resource")

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-linodecluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=linodeclusters,versions=v1alpha4,name=validation.linodecluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha4-linodecluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=linodeclusters,versions=v1alpha4,name=default.linodecluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &LinodeCluster{}
	_ webhook.Validator = &LinodeCluster{}
)

func (r *LinodeCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LinodeCluster) Default() {}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *LinodeCluster) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeCluster) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	oldLinodeCluster, ok := old.(*LinodeCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected an LinodeCluster but got a %T", old))
	}

	if r.Spec.Region != oldLinodeCluster.Spec.Region {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "region"), r.Spec.Region, "field is immutable"))
	}

	if !reflect.DeepEqual(clusterv1.APIEndpoint{}, oldLinodeCluster.Spec.ControlPlaneEndpoint) && !reflect.DeepEqual(r.Spec.ControlPlaneEndpoint, oldLinodeCluster.Spec.ControlPlaneEndpoint) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controlPlaneEndpoint"), r.Spec.Region, "field is immutable"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeCluster) ValidateDelete() error {
	return nil
}
