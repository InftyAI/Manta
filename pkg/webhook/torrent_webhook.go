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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/inftyai/manta/api/v1alpha1"
)

type TorrentWebhook struct{}

// SetupTorrentWebhook will setup the manager to manage the webhooks
func SetupTorrentWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Torrent{}).
		WithDefaulter(&TorrentWebhook{}).
		WithValidator(&TorrentWebhook{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-manta-io-v1alpha1-torrent,mutating=true,failurePolicy=fail,sideEffects=None,groups=manta.io,resources=torrents,verbs=create;update,versions=v1alpha1,name=mtorrent.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &TorrentWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *TorrentWebhook) Default(ctx context.Context, obj runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-manta-io-v1alpha1-torrent,mutating=false,failurePolicy=fail,sideEffects=None,groups=manta.io,resources=torrents,verbs=create;update,versions=v1alpha1,name=vtorrent.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &TorrentWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *TorrentWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	allErrs := w.generateValidate(obj)
	return nil, allErrs.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *TorrentWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	old := oldObj.(*api.Torrent)
	new := newObj.(*api.Torrent)

	var allErrs field.ErrorList
	specPath := field.NewPath("spec")
	if *old.Spec.Preheat && !*new.Spec.Preheat {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("preheat"), "preheat can only be transitioned from false to true"))
	}
	allErrs = append(allErrs, w.generateValidate(newObj)...)
	return nil, allErrs.ToAggregate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *TorrentWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *TorrentWebhook) generateValidate(obj runtime.Object) field.ErrorList {
	torrent := obj.(*api.Torrent)
	specPath := field.NewPath("spec")

	var allErrs field.ErrorList
	if torrent.Spec.Hub == nil {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("hub"), "hub can't be null"))
	}

	if !(torrent.Spec.TTLSecondsAfterReady == nil || *torrent.Spec.TTLSecondsAfterReady == time.Duration(0)) {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("ttlSecondsAfterReady"), "only support nil and 0 right now"))
	}

	return allErrs
}
