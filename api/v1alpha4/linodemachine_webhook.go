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
	"reflect"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var _ = logf.Log.WithName("linodemachine-resource")

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-linodemachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=linodemachines,versions=v1alpha4,name=validation.linodemachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha4-linodemachine,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=linodemachines,versions=v1alpha4,name=default.linodemachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &LinodeMachine{}
	_ webhook.Validator = &LinodeMachine{}
)

func (r *LinodeMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LinodeMachine) Default() {}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *LinodeMachine) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeMachine) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	newLinodeMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
	if err != nil {
		return apierrors.NewInternalError(errors.Wrap(err, "failed to convert new LinodeMachine to unstructured object"))
	}

	oldLinodeMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(old)
	if err != nil {
		return apierrors.NewInternalError(errors.Wrap(err, "failed to convert old LinodeMachine to unstructured object"))
	}

	newLinodeMachineSpec := newLinodeMachine["spec"].(map[string]interface{})
	oldLinodeMachineSpec := oldLinodeMachine["spec"].(map[string]interface{})

	// allow changes to providerID
	delete(oldLinodeMachineSpec, "providerID")
	delete(newLinodeMachineSpec, "providerID")

	// allow changes to additionalTags
	delete(oldLinodeMachineSpec, "additionalTags")
	delete(newLinodeMachineSpec, "additionalTags")

	if !reflect.DeepEqual(oldLinodeMachineSpec, newLinodeMachineSpec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeMachine) ValidateDelete() error {
	return nil
}
