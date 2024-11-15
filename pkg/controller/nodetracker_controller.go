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
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher"
)

// NodeTrackerReconciler reconciles a NodeTracker object
type NodeTrackerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	dispatcher *dispatcher.Dispatcher
}

func NewNodeTrackerReconciler(client client.Client, scheme *runtime.Scheme, dispatcher *dispatcher.Dispatcher) *NodeTrackerReconciler {
	return &NodeTrackerReconciler{
		Client:     client,
		Scheme:     scheme,
		dispatcher: dispatcher,
	}
}

//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=manta.io,resources=nodetrackers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *NodeTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile NodeTracker", "NodeTracker", req.Name)

	nodeTracker := &api.NodeTracker{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, nodeTracker); err != nil {
		// If node trigger events arrived, but nodeTracker doesn't exist, this may
		// because agent doesn't deployed on this node, so nodeTracker doesn't exist.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	node := &corev1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, node); err != nil {
		// Work for integration test.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !reflect.DeepEqual(node.Labels, nodeTracker.Labels) {
		nodeTracker.Labels = node.Labels
		if err := r.Client.Update(ctx, nodeTracker); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NodeTrackerReconciler) Create(e event.CreateEvent) bool {
	nodeTracker, match := e.Object.(*api.NodeTracker)
	if !match {
		return false
	}

	r.dispatcher.AddNodeTracker(nodeTracker)
	return true
}

func (r *NodeTrackerReconciler) Update(e event.UpdateEvent) bool {
	newObj, match := e.ObjectNew.(*api.NodeTracker)
	// Other objs like Nodes should not be handled below.
	if !match {
		return true
	}

	oldObj := e.ObjectOld.(*api.NodeTracker)
	r.dispatcher.UpdateNodeTracker(oldObj, newObj)
	return true
}

func (r *NodeTrackerReconciler) Delete(e event.DeleteEvent) bool {
	obj, match := e.Object.(*api.NodeTracker)
	if !match {
		return false
	}

	r.dispatcher.DeleteNodeTracker(obj)
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
		Watches(&corev1.Node{}, &handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool { return false },
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldNode := e.ObjectOld.(*corev1.Node)
					newNode := e.ObjectNew.(*corev1.Node)
					return !reflect.DeepEqual(oldNode.Labels, newNode.Labels)

				},
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}
