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
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	consts "github.com/inftyai/manta/api"
	api "github.com/inftyai/manta/api/v1alpha1"
	defaults "github.com/inftyai/manta/pkg"
)

type PodWebhook struct{}

// SetupPodWebhook will setup the manager to manage the webhooks
func SetupPodWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodWebhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mpod.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &PodWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *PodWebhook) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	// This should not happen since filtered by webhook objectSelector.
	if pod.Labels == nil || pod.Labels[api.TorrentNameLabelKey] == "" {
		return nil
	}
	port, err := strconv.Atoi(consts.HttpPort)
	if err != nil {
		return err
	}

	initContainer := corev1.Container{
		Name:    defaults.PREHEAT_CONTAINER_NAME,
		Image:   defaults.PREHEAT_IMAGE,
		Command: []string{"/manager"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: int32(port),
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)
	return nil
}
