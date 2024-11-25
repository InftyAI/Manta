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

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ReplicationReconciler reconciles a Replication object
type ReplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewReplicationReconciler(client client.Client, scheme *runtime.Scheme) *ReplicationReconciler {
	return &ReplicationReconciler{
		Client: client,
		Scheme: scheme,
	}
}

//+kubebuilder:rbac:groups=manta.io,resources=replications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=manta.io,resources=replications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=manta.io,resources=replications/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	replication := &api.Replication{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, replication); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("reconcile Replication", "Replication", klog.KObj(replication))

	// Leave the left reconciliation to agent controller.
	if setReplicationCondition(replication) {
		return ctrl.Result{}, r.Status().Update(ctx, replication)
	}

	return ctrl.Result{}, nil
}

// Only watch for create events.
func (r *ReplicationReconciler) Create(e event.CreateEvent) bool {
	return true
}

func (r *ReplicationReconciler) Delete(e event.DeleteEvent) bool {
	return false
}

func (r *ReplicationReconciler) Update(e event.UpdateEvent) bool {
	return false
}

func (r *ReplicationReconciler) Generic(e event.GenericEvent) bool {
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Replication{}).
		WithEventFilter(r).
		Complete(r)
}

func setReplicationCondition(replication *api.Replication) (changed bool) {
	if len(replication.Status.Conditions) == 0 {
		condition := metav1.Condition{
			Type:    api.PendingConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: "Waiting for downloading",
		}
		replication.Status.Phase = ptr.To[string](api.PendingConditionType)
		return apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
	}
	return false
}
