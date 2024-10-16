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
	"fmt"
	"os"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	agenthandler "github.com/inftyai/manta/agent/pkg/handler"
	api "github.com/inftyai/manta/api/v1alpha1"
)

var (
	NODE_NAME = os.Getenv("NODE_NAME")
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

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	replication := &api.Replication{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, replication); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Filter out unrelated events.
	if replication.Spec.NodeName != NODE_NAME || replicationReady(replication) || replication.DeletionTimestamp != nil {
		logger.V(10).Info("Skip replication", "Replication", klog.KObj(replication))
		return ctrl.Result{}, nil
	}

	logger.Info("Reconcile replication", "Replication", klog.KObj(replication))

	if conditionChanged := setReplicationCondition(replication, api.DownloadConditionType); conditionChanged {
		return ctrl.Result{}, r.Status().Update(ctx, replication)
	}

	// This may take a long time, the concurrency is controlled by the MaxConcurrentReconciles.
	succeeded, stateChanged := agenthandler.HandleReplication(logger, r.Client, replication)
	if stateChanged {
		// TODO: using patch to avoid update conflicts.
		if err := r.Update(ctx, replication); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !succeeded {
		return ctrl.Result{}, fmt.Errorf("handle Replication error")
	}

	if tuplesReady(replication) {
		// If succeeded, set to ready.
		if conditionChanged := setReplicationCondition(replication, api.ReadyConditionType); conditionChanged {
			return ctrl.Result{}, r.Status().Update(ctx, replication)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Replication{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}

func setReplicationCondition(replication *api.Replication, conditionType string) (changed bool) {
	if conditionType == api.DownloadConditionType {
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Downloading",
			Message: "Downloading chunks",
		}
		return apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
	}

	if conditionType == api.ReadyConditionType {
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Download chunks successfully",
		}
		return apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
	}

	return false
}

func replicationReady(replication *api.Replication) bool {
	return apimeta.IsStatusConditionTrue(replication.Status.Conditions, api.ReadyConditionType)
}

func tuplesReady(replication *api.Replication) bool {
	for _, tuple := range replication.Spec.Tuples {
		if *tuple.State != api.FinishedStateType {
			return false
		}
	}
	return true
}
