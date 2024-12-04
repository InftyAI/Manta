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

package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/inftyai/manta/api/v1alpha1"
)

// PodReconciler reconciles a Torrent object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewPodReconciler(client client.Client, scheme *runtime.Scheme) *PodReconciler {
	return &PodReconciler{
		Client: client,
		Scheme: scheme,
	}
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("reconcile Pod")
	// This should not happen, double check here.
	if pod.Labels == nil || pod.Labels[api.TorrentNameLabelKey] == "" {
		return ctrl.Result{}, nil
	}

	torrentName := pod.Labels[api.TorrentNameLabelKey]

	torrent := &api.Torrent{}
	if err := r.Get(ctx, types.NamespacedName{Name: torrentName}, torrent); err != nil {
		return ctrl.Result{}, err
	}

	newTorrent := constructTorrent(torrent, pod)
	if err := r.Client.Create(ctx, &newTorrent); err != nil {
		logger.Error(err, "failed to create Torrent", "Torrent", klog.KObj(&newTorrent))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PodReconciler) Create(e event.CreateEvent) bool {
	pod, match := e.Object.(*corev1.Pod)
	if !match {
		return false
	}

	// Pod should be managed by Manta.
	if pod.Labels == nil || pod.Labels[api.TorrentNameLabelKey] == "" {
		return false
	}

	return true
}

func (r *PodReconciler) Update(e event.UpdateEvent) bool {
	return false
}

func (r *PodReconciler) Delete(e event.DeleteEvent) bool {
	return false
}

func (r *PodReconciler) Generic(e event.GenericEvent) bool {
	return false
}

func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(r).
		Complete(r)
}

func constructTorrent(torrent *api.Torrent, pod *corev1.Pod) api.Torrent {
	newTorrent := api.Torrent{}
	newTorrent.ObjectMeta.Name = torrent.Name + "--tmp--" + pod.Spec.NodeName
	newTorrent.TypeMeta = torrent.TypeMeta
	newTorrent.Annotations = map[string]string{api.ParentPodNameAnnoKey: pod.Namespace + "/" + pod.Name}
	newTorrent.Spec = torrent.Spec
	newTorrent.Spec.Preheat = ptr.To[bool](true)
	newTorrent.Spec.Replicas = ptr.To[int32](1)
	newTorrent.Spec.ReclaimPolicy = ptr.To[api.ReclaimPolicy](api.RetainReclaimPolicy)
	newTorrent.Spec.TTLSecondsAfterReady = ptr.To[time.Duration](0)
	newTorrent.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": pod.Spec.NodeName}

	return newTorrent
}
