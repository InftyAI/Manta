/*
Copyright 2024.

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

package webhook

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/inftyai/manta/api/v1alpha1"
)

type ReplicationWebhook struct{}

// SetupTorrentWebhook will setup the manager to manage the webhooks
func SetupReplicationWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Replication{}).
		WithDefaulter(&ReplicationWebhook{}).
		WithValidator(&ReplicationWebhook{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-manta-io-v1alpha1-replication,mutating=true,failurePolicy=fail,sideEffects=None,groups=manta.io,resources=replications,verbs=create;update,versions=v1alpha1,name=mreplication.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ReplicationWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *ReplicationWebhook) Default(ctx context.Context, obj runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-manta-io-v1alpha1-replication,mutating=false,failurePolicy=fail,sideEffects=None,groups=manta.io,resources=replications,verbs=create;update,versions=v1alpha1,name=vreplication.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ReplicationWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *ReplicationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	allErrs := w.generateValidate(obj)
	return nil, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *ReplicationWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	allErrs := w.generateValidate(newObj)
	return nil, allErrs.ToAggregate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *ReplicationWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *ReplicationWebhook) generateValidate(obj runtime.Object) field.ErrorList {
	replication := obj.(*api.Replication)
	specPath := field.NewPath("spec")

	var allErrs field.ErrorList

	if replication.Spec.Destination != nil && replication.Spec.Destination.Hub == nil && replication.Spec.Destination.URI == nil {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("destination"), "hub and URI couldn't be both null in Destination"))
	}
	if replication.Spec.Source.Hub == nil && replication.Spec.Source.URI == nil {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("source"), "hub and URI couldn't be both null in Source"))
	}
	if replication.Spec.Source.URI != nil {
		splits := strings.Split(*replication.Spec.Source.URI, "://")
		if splits[0] == "localhost" && replication.Spec.Destination != nil {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("destination"), "destination must be nil once source is localhost"))
		}
	}
	if replication.Spec.Source.Hub != nil {
		if replication.Spec.Destination == nil || replication.Spec.Destination.URI == nil {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("destination.uri"), "destination.uri must not be nil once source.hub is not nil"))
		}
		// TODO: we may support upload to remote store in the future if highly demanded.
		splits := strings.Split(*replication.Spec.Destination.URI, "://")
		if splits[0] != "localhost" {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("destination.uri"), "destination.uri must be localhost once source.hub is not nil"))
		}
	}
	return allErrs
}
