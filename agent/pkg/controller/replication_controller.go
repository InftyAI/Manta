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
	"os"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
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

// Agent Replication reconciler only focuses on downloading and replicating process, not interested in the
// Replication lifecycle management.
func (r *ReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	replication := &api.Replication{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, replication); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Filter out unrelated events.
	if replication.Spec.NodeName != NODE_NAME ||
		replicationReady(replication) ||
		len(replication.Status.Conditions) == 0 {
		logger.V(10).Info("Skip replication", "Replication", klog.KObj(replication))
		return ctrl.Result{}, nil
	}

	logger.Info("Reconcile replication", "Replication", klog.KObj(replication))

	if conditionChanged := setReplicationCondition(replication, api.DownloadConditionType); conditionChanged {
		return ctrl.Result{}, r.Status().Update(ctx, replication)
	}

	// This may take a long time, the concurrency is controlled by the MaxConcurrentReconciles.
	if err := agenthandler.HandleReplication(logger, replication); err != nil {
		return ctrl.Result{}, err
	} else {
		if conditionChanged := setReplicationCondition(replication, api.ReadyConditionType); conditionChanged {
			if err := r.Status().Update(ctx, replication); err != nil {
				return ctrl.Result{}, err
			}
		}
		if err := r.updateNodeTracker(ctx, replication); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReplicationReconciler) updateNodeTracker(ctx context.Context, replication *api.Replication) error {
	nodeTracker := &api.NodeTracker{}
	if err := r.Get(ctx, types.NamespacedName{Name: replication.Spec.NodeName}, nodeTracker); err != nil {
		return err
	}

	chunkName := replication.Spec.ChunkName
	for _, chunk := range nodeTracker.Spec.Chunks {
		if chunk.ChunkName == chunkName {
			// Already included.
			return nil
		}
	}
	nodeTracker.Spec.Chunks = append(nodeTracker.Spec.Chunks, api.ChunkTracker{
		ChunkName: chunkName,
		SizeBytes: replication.Spec.SizeBytes,
	})
	if err := r.Client.Update(ctx, nodeTracker); err != nil {
		return err
	}
	return nil
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
		replication.Status.Phase = ptr.To[string](api.DownloadConditionType)
		return apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
	}

	if conditionType == api.ReadyConditionType {
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Download chunks successfully",
		}
		replication.Status.Phase = ptr.To[string](api.ReadyConditionType)
		return apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
	}

	return false
}

func replicationReady(replication *api.Replication) bool {
	return apimeta.IsStatusConditionTrue(replication.Status.Conditions, api.ReadyConditionType)
}
