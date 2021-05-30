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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var _ = logf.Log.WithName("linodemachinetemplate-resource")

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-linodemachinetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=linodemachinetemplates,versions=v1alpha4,name=validation.linodemachinetemplate.infrastructure.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Validator = &LinodeMachineTemplate{}
)

func (r *LinodeMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeMachineTemplate) ValidateCreate() error {
	var allErrs field.ErrorList
	spec := r.Spec.Template.Spec

	if spec.ProviderID != nil {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "template", "spec", "providerID"), "cannot be set in templates"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeMachineTemplate) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	oldLinodeMachineTemplate := old.(*LinodeMachineTemplate)
	if !reflect.DeepEqual(r.Spec, oldLinodeMachineTemplate.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "LinodeMachineTemplateSpec is immutable"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LinodeMachineTemplate) ValidateDelete() error {
	return nil
}
