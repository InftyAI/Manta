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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cons "github.com/inftyai/manta/api"
	api "github.com/inftyai/manta/api/v1alpha1"
	defaults "github.com/inftyai/manta/pkg"
	"github.com/inftyai/manta/pkg/dispatcher"
	"github.com/inftyai/manta/pkg/util"
)

// TorrentReconciler reconciles a Torrent object
type TorrentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	dispatcher *dispatcher.Dispatcher
}

func NewTorrentReconciler(client client.Client, scheme *runtime.Scheme, dispatcher *dispatcher.Dispatcher) *TorrentReconciler {
	return &TorrentReconciler{
		Client:     client,
		Scheme:     scheme,
		dispatcher: dispatcher,
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

	logger.Info("reconcile Torrent")

	// Noe need to handle Torrent at this point just because we don't want to
	// download the files.
	if torrent.Spec.Preheat != nil && !*torrent.Spec.Preheat {
		return ctrl.Result{}, nil
	}

	// TODO: delete torrent at anytime.
	if torrentReady(torrent) && torrentDeleting(torrent) {
		logger.Info("start to handle torrent deletion")

		if err := r.handleDeletion(ctx, torrent); err != nil {
			logger.Error(err, "failed to handle deletion")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if torrentReady(torrent) {
		logger.Info("start to handle torrent ready")

		if err := r.handleReady(ctx, torrent); err != nil {
			logger.Error(err, "failed to handle ready status", "Torrent", klog.KObj(torrent))
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if torrent.Status.Repo == nil {
		logger.Info("start to handle torrent creation")

		if err := r.handleCreation(ctx, torrent); err != nil {
			logger.Error(err, "failed to handle creation")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	nodeTrackers := &api.NodeTrackerList{}
	if err := r.List(ctx, nodeTrackers, &client.ListOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	// handleDispatcher should be idempotent.
	torrentStatusChanged, err := r.handleDispatcher(ctx, torrent, nodeTrackers.Items)
	if err != nil {
		logger.Error(err, "failed to dispatcher torrent")
		return ctrl.Result{}, err
	}

	replications, err := r.replications(ctx, torrent)
	if err != nil {
		return ctrl.Result{}, err
	}

	// set the condition.
	conditionChanged := setTorrentCondition(torrent, replications)
	if torrentStatusChanged || conditionChanged {
		return ctrl.Result{}, r.Status().Update(ctx, torrent)
	}

	return ctrl.Result{}, nil
}

func (r *TorrentReconciler) handleCreation(ctx context.Context, torrent *api.Torrent) (err error) {
	if controllerutil.AddFinalizer(torrent, api.TorrentProtectionFinalizer) {
		if err := r.Client.Update(ctx, torrent); err != nil {
			return err
		}
	}

	// We'll get the latest torrent, update the status will not lead to conflict most of the time.

	_ = setTorrentCondition(torrent, nil)

	// TODO: We only support hub right now, we need to support spec.URI in the future as well.
	objects, err := util.ListRepoObjects(torrent.Spec.Hub.RepoID, *torrent.Spec.Hub.Revision)
	if err != nil {
		return err
	}
	constructRepoStatus(torrent, objects)

	return r.Client.Status().Update(ctx, torrent)
}

func (r *TorrentReconciler) handleDeletion(ctx context.Context, torrent *api.Torrent) error {
	if *torrent.Spec.ReclaimPolicy == api.RetainReclaimPolicy {
		if controllerutil.RemoveFinalizer(torrent, api.TorrentProtectionFinalizer) {
			if err := r.Client.Update(ctx, torrent); err != nil {
				return err
			}
		}
		return nil
	}

	if *torrent.Spec.ReclaimPolicy == api.DeleteReclaimPolicy {
		replications, statusChanged, err := r.dispatcher.ReclaimReplications(ctx, torrent)
		if err != nil {
			return err
		}

		for _, rep := range replications {
			if err := r.Client.Create(ctx, rep); err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}

		if statusChanged || setTorrentCondition(torrent, nil) {
			return r.Status().Update(ctx, torrent)
		}

		replicationList, err := r.replications(ctx, torrent)
		if err != nil {
			return err
		}

		if replicationsReady(replicationList) {
			if controllerutil.RemoveFinalizer(torrent, api.TorrentProtectionFinalizer) {
				if err := r.Client.Update(ctx, torrent); err != nil {
					return err
				}
				return nil
			}
		}

	}
	return nil
}

func (r *TorrentReconciler) handleReady(ctx context.Context, torrent *api.Torrent) error {
	// request the callback to notify the pod, model download/sync is finished.
	if torrent.Annotations[api.ParentPodNameAnnoKey] != "" {
		if err := callback(ctx, r.Client, torrent); err != nil {
			return err
		}
	}

	// TODO: once ttl supports other values than 0, we need to refactor here.
	if torrent.Spec.TTLSecondsAfterReady != nil && *torrent.Spec.TTLSecondsAfterReady == time.Duration(0) {
		// Corresponding Replications will be deleted as well.
		if err := r.Client.Delete(ctx, torrent); err != nil {
			return err
		}
		return nil
	}

	replications, err := r.replications(ctx, torrent)
	if err != nil {
		return err
	}

	for _, replication := range replications {
		if err := r.Client.Delete(ctx, &replication); err != nil {
			return err
		}
	}
	return nil
}

func (r *TorrentReconciler) handleDispatcher(ctx context.Context, torrent *api.Torrent, nodeTrackers []api.NodeTracker) (statusChanged bool, err error) {
	// Do not delete the Replication manually or they will be created again.
	replications, statusChanged, firstTime, err := r.dispatcher.PrepareReplications(ctx, torrent, nodeTrackers)
	if err != nil {
		return false, err
	}

	// We may have no replication to create, e.g. all the chunks are replicated.
	// Set the Torrent to ready directly.
	if len(replications) == 0 && firstTime {
		condition := metav1.Condition{
			Type:    api.ReadyConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "All chunks are replicated already",
		}
		if setTorrentConditionTo(torrent, condition) {
			return false, r.Status().Update(ctx, torrent)
		}
	}

	for _, rep := range replications {
		// If Replication is duplicated, just ignore here.
		if err := r.Client.Create(ctx, rep); err != nil && !apierrors.IsAlreadyExists(err) {
			return false, err
		}
	}
	return statusChanged, nil
}

func (r *TorrentReconciler) replications(ctx context.Context, torrent *api.Torrent) ([]api.Replication, error) {
	replicationList := api.ReplicationList{}
	selector := labels.SelectorFromSet(labels.Set{api.TorrentNameLabelKey: torrent.Name})
	if err := r.List(ctx, &replicationList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return nil, err
	}
	return replicationList.Items, nil
}

func (r *TorrentReconciler) Create(e event.CreateEvent) bool {
	_, match := e.Object.(*api.Torrent)
	return match
}

func (r *TorrentReconciler) Update(e event.UpdateEvent) bool {
	return true
}

func (r *TorrentReconciler) Delete(e event.DeleteEvent) bool {
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
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}

func setTorrentCondition(torrent *api.Torrent, replications []api.Replication) (changed bool) {
	// Set to Pending condition.
	if torrent.Status.Repo == nil {
		condition := metav1.Condition{
			Type:    api.PendingConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: "Waiting for Replication creations",
		}
		return setTorrentConditionTo(torrent, condition)
	}

	// Set to Reclaiming condition.
	if torrentReady(torrent) && torrentDeleting(torrent) {
		condition := metav1.Condition{
			Type:    api.ReclaimingConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Reclaiming",
			Message: "Deleting chunks",
		}
		return setTorrentConditionTo(torrent, condition)
	}

	if torrentReady(torrent) {
		return false
	}

	// Set to Ready condition.
	if apimeta.IsStatusConditionTrue(torrent.Status.Conditions, api.ReplicateConditionType) && replicationsReady(replications) {
		condition := metav1.Condition{
			Type:    api.ReadyConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Chunks replicated successfully",
		}
		return setTorrentConditionTo(torrent, condition)
	}

	// Set to Replicating condition.
	if torrentDownloading(replications) {
		condition := metav1.Condition{
			Type:    api.ReplicateConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "Replicating",
			Message: "Replicating chunks",
		}
		return setTorrentConditionTo(torrent, condition)
	}

	return false
}

func setTorrentConditionTo(torrent *api.Torrent, condition metav1.Condition) (changed bool) {
	torrent.Status.Phase = ptr.To[string](condition.Type)
	return apimeta.SetStatusCondition(&torrent.Status.Conditions, condition)
}

func torrentDownloading(replications []api.Replication) bool {
	for _, replication := range replications {
		// If one replication is in replicating, then yes.
		if apimeta.IsStatusConditionTrue(replication.Status.Conditions, api.ReplicateConditionType) {
			return true
		}
	}
	return false
}

func replicationsReady(replications []api.Replication) bool {
	for _, obj := range replications {
		if !apimeta.IsStatusConditionTrue(obj.Status.Conditions, api.ReadyConditionType) {
			return false
		}
	}
	return true
}

func torrentReady(torrent *api.Torrent) bool {
	return apimeta.IsStatusConditionTrue(torrent.Status.Conditions, api.ReadyConditionType)
}

func torrentDeleting(torrent *api.Torrent) bool {
	return !torrent.DeletionTimestamp.IsZero()
}

// We have one chunk for one file for now.
func constructRepoStatus(torrent *api.Torrent, objects []*util.ObjectBody) {
	repo := &api.RepoStatus{}

	if torrent.Spec.Hub.Filename != nil {
		// The repo could contain multiple objects(files) in the same directory, but
		// we only need one file.
		for _, obj := range objects {
			if obj.Path == *torrent.Spec.Hub.Filename {
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

func callback(ctx context.Context, cli client.Client, torrent *api.Torrent) error {
	splits := strings.Split(torrent.Annotations[api.ParentPodNameAnnoKey], "/")
	if len(splits) != 2 {
		return errors.New("namespaced name is not right")
	}

	pod := corev1.Pod{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: splits[0], Name: splits[1]}, &pod); err != nil {
		return err
	}

	// Once invoked callback, no longer to call again.
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Name == defaults.PREHEAT_CONTAINER_NAME {
			if status.Ready {
				return nil
			}
		}
	}

	url := fmt.Sprintf("http://%s:%s/preheated", pod.Status.PodIP, cons.HttpPort)

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return errors.New("status not right")
	}

	return nil
}
