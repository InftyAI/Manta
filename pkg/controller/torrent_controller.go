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
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher"
	"github.com/inftyai/manta/pkg/util"
)

// TorrentReconciler reconciles a Torrent object
type TorrentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Record     record.EventRecorder
	Dispatcher *dispatcher.Dispatcher
}

func NewTorrentReconciler(client client.Client, scheme *runtime.Scheme, record record.EventRecorder, dispatcher *dispatcher.Dispatcher) *TorrentReconciler {
	return &TorrentReconciler{
		Client:     client,
		Scheme:     scheme,
		Record:     record,
		Dispatcher: dispatcher,
	}
}

//+kubebuilder:rbac:groups=manta.io,resources=torrents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=manta.io,resources=torrents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=manta.io,resources=torrents/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *TorrentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	torrent := &api.Torrent{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, torrent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("reconcile Torrent", "Torrent", klog.KObj(torrent))

	// Once ready, no need to do anything.
	if torrentReady(torrent) {
		return ctrl.Result{}, nil
	}

	// Handle Pending status.
	if torrent.Status.Repo == nil {
		_ = setTorrentCondition(torrent, nil)

		// TODO: We only support modelHub right now, we need to support spec.URI in the future as well.
		objects, err := util.ListRepoObjects(torrent.Spec.ModelHub.ModelID, *torrent.Spec.ModelHub.Revision)
		if err != nil {
			return ctrl.Result{}, err
		}
		constructRepoOfStatus(torrent, objects)

		return ctrl.Result{}, r.Client.Status().Update(ctx, torrent)
	}

	// Handle dispatch.

	// Do not delete the Replication manually or they will be created again.
	replications, torrentStatusChanged, err := r.Dispatcher.PrepareReplications(torrent)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(replications) > 0 {
		for _, rep := range replications {
			// If Replication is duplicated, just ignore here.
			if err := r.Client.Create(ctx, rep); err != nil && !errors.IsAlreadyExists(err) {
				return ctrl.Result{}, err
			}
		}
	}

	replicationList := api.ReplicationList{}
	selector := labels.SelectorFromSet(labels.Set{api.TorrentNameLabelKey: torrent.Name})
	if err := r.List(ctx, &replicationList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return ctrl.Result{}, err
	}

	conditionChanged := setTorrentCondition(torrent, &replicationList)
	if torrentStatusChanged || conditionChanged {
		return ctrl.Result{}, r.Status().Update(ctx, torrent)
	}

	return ctrl.Result{}, nil
}

func (r *TorrentReconciler) Create(e event.CreateEvent) bool {
	torrent, match := e.Object.(*api.Torrent)
	if !match {
		return false
	}

	logger := log.FromContext(context.Background()).WithValues("Torrent", klog.KObj(torrent))
	logger.V(2).Info("Torrent create event")

	return true
}

func (r *TorrentReconciler) Delete(e event.DeleteEvent) bool {
	return true
}

func (r *TorrentReconciler) Update(e event.UpdateEvent) bool {
	return true
}

func (r *TorrentReconciler) Generic(e event.GenericEvent) bool {
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *TorrentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapFunc := func(ctx context.Context, obj client.Object) []ctrl.Request {
		labels := obj.GetLabels()
		if labels == nil {
			return nil
		}

		value := labels[api.TorrentNameLabelKey]
		if value == "" {
			return nil
		}

		return []ctrl.Request{
			{NamespacedName: types.NamespacedName{Name: value}},
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Torrent{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Watches(&api.Replication{}, handler.EnqueueRequestsFromMapFunc(mapFunc),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool { return false },
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldRep := e.ObjectOld.(*api.Replication)
					newRep := e.ObjectNew.(*api.Replication)
					return !reflect.DeepEqual(oldRep.Spec.Tuples, newRep.Spec.Tuples)
				},
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}

func setTorrentCondition(torrent *api.Torrent, replicationList *api.ReplicationList) (changed bool) {
	if torrent.Status.Repo == nil {
		condition := metav1.Condition{
			Type:    api.PendingConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: "Waiting for Replication creations",
		}
		return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
	}

	if torrentDownloading(replicationList) {
		condition := metav1.Condition{
			Type:    api.DownloadConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Downloading",
			Message: "Downloading chunks",
		}
		return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
	}

	if replicationsReady(replicationList) {
		condition := metav1.Condition{
			Type:    api.ReadyConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Download chunks successfully",
		}
		return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
	}

	return false
}

func torrentDownloading(replicationList *api.ReplicationList) bool {
	return replicationList != nil && len(replicationList.Items) > 0
}

func replicationsReady(replicationList *api.ReplicationList) bool {
	if len(replicationList.Items) == 0 {
		return false
	}

	for _, obj := range replicationList.Items {
		if apimeta.IsStatusConditionFalse(obj.Status.Conditions, api.ReadyConditionType) {
			return false
		}
	}
	return true
}

func torrentReady(torrent *api.Torrent) bool {
	return apimeta.IsStatusConditionTrue(torrent.Status.Conditions, api.ReadyConditionType)
}

// We have one chunk for one file for now.
func constructRepoOfStatus(torrent *api.Torrent, objects []*util.ObjectBody) {
	repo := &api.RepoStatus{}

	if torrent.Spec.ModelHub.Filename != nil {
		// The repo could contain multiple objects(files) in the same directory, but
		// we only need one file.
		for _, obj := range objects {
			if obj.Path == *torrent.Spec.ModelHub.Filename {
				chunks := []api.ChunkStatus{}
				chunks = append(chunks, api.ChunkStatus{
					// TODO: Each file only has one chunk for now.
					Name:      fmt.Sprintf("%s--0001", obj.Oid),
					State:     api.PendingTrackerState,
					SizeBytes: obj.Size,
				})
				repo.Objects = []api.ObjectStatus{
					{
						Path:   obj.Path,
						Type:   api.ObjectType(obj.Type),
						Chunks: chunks,
					},
				}
				break
			}
		}
	} else {
		for _, obj := range objects {
			chunks := []api.ChunkStatus{}
			chunks = append(chunks, api.ChunkStatus{
				// TODO: Each file only has one chunk for now.
				Name:      fmt.Sprintf("%s--0001", obj.Oid),
				State:     api.PendingTrackerState,
				SizeBytes: obj.Size,
			})
			repo.Objects = append(repo.Objects, api.ObjectStatus{
				Path:   obj.Path,
				Type:   api.ObjectType(obj.Type),
				Chunks: chunks,
			})
		}
	}
	torrent.Status.Repo = repo
}
