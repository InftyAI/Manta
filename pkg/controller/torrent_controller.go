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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

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

func repoName(modelID string) string {
	return "models--" + strings.ReplaceAll(modelID, "/", "--")
}

// We have one chunk for one file for now.
func constructRepoOfStatus(torrent *api.Torrent, objects []*util.ObjectBody) {
	repo := &api.RepoStatus{}

	if torrent.Spec.ModelHub.Filename != nil {
		for _, obj := range objects {
			if obj.Path == *torrent.Spec.ModelHub.Filename {
				repo.Objects = []*api.ObjectStatus{
					{
						Path: obj.Path,
						Type: api.ObjectType(obj.Type),
						Chunks: []*api.ChunkStatus{
							{
								// Each file only has one chunk right now.
								Name:      obj.Oid + "--0001",
								State:     api.PendingTrackerState,
								SizeBytes: obj.Size,
							},
						},
					},
				}
				break
			}
		}
	} else {
		repoName := repoName(torrent.Spec.ModelHub.ModelID)
		repo.Name = &repoName
		for _, obj := range objects {
			repo.Objects = append(repo.Objects, &api.ObjectStatus{
				Path: obj.Path,
				Type: api.ObjectType(obj.Type),
				Chunks: []*api.ChunkStatus{
					{
						// Each file only has one chunk right now.
						Name:      obj.Oid + "--0001",
						State:     api.PendingTrackerState,
						SizeBytes: obj.Size,
					},
				},
			})
		}
	}
	torrent.Status.Repo = repo
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

	logger.V(10).Info("reconcile Torrent", "Torrent", klog.KObj(torrent))

	// Handle Pending status.
	if torrent.Status.Repo == nil {
		_ = setCondition(torrent)

		// TODO: We only support modelHub right now, we need to support spec.URI in the future as well.
		objects, err := util.ListRepoObjects(torrent.Spec.ModelHub.ModelID, *torrent.Spec.ModelHub.Revision)
		if err != nil {
			return ctrl.Result{}, err
		}
		constructRepoOfStatus(torrent, objects)

		return ctrl.Result{}, r.Client.Status().Update(ctx, torrent)
	}

	// Handle dispatch.

	replications, torrentStatusChanged, err := r.Dispatcher.PrepareReplications(torrent)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, rep := range replications {
		if err := r.Patch(ctx, rep, client.Apply, &client.PatchOptions{
			FieldManager: "Torrent",
		}); err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	conditionChanged := setCondition(torrent)
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Torrent{}).
		Complete(r)
}

func setCondition(torrent *api.Torrent) (changed bool) {
	if torrent.Status.Repo == nil {
		condition := metav1.Condition{
			Type:    api.PendingConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: "Waiting for Replication creations",
		}
		return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
	}

	if torrentReady(torrent) {
		condition := metav1.Condition{
			Type:    api.ReadyConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Download chunks successfully",
		}
		return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
	}

	condition := metav1.Condition{
		Type:    api.DownloadConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "Downloading",
		Message: "Downloading chunks",
	}
	return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
}

func torrentReady(torrent *api.Torrent) bool {
	if torrent.Status.Repo == nil {
		return false
	}

	for _, obj := range torrent.Status.Repo.Objects {
		for _, chunk := range obj.Chunks {
			if chunk.State != api.ReadyTrackerState {
				return false
			}
		}
	}
	return true
}
