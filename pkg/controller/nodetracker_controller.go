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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/inftyai/manta/api/v1alpha1"
)

// NodeTrackerReconciler reconciles a NodeTracker object
type NodeTrackerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *NodeTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile NodeTracker", "NodeTracker", req.Name)
	return ctrl.Result{}, nil
}

func (r *NodeTrackerReconciler) Create(e event.CreateEvent) bool {
	// TODO: update cache
	return true
}

func (r *NodeTrackerReconciler) Update(e event.UpdateEvent) bool {
	// TODO: update cache
	return true
}

func (r *NodeTrackerReconciler) Delete(e event.DeleteEvent) bool {
	// TODO: update cache
	return true
}

func (r *NodeTrackerReconciler) Generic(e event.GenericEvent) bool {
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.NodeTracker{}).
		WithEventFilter(r).
		Complete(r)
}
