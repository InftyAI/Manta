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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/util"
)

// TorrentReconciler reconciles a Torrent object
type TorrentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewTorrentReconciler(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) *TorrentReconciler {
	return &TorrentReconciler{
		Client: client,
		Scheme: scheme,
		Record: record,
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

	logger.V(10).Info("reconcile Torrent", "Torrent", klog.KObj(torrent))

	// Handle Pending status.
	if len(torrent.Status.Files) == 0 {
		if torrent.Spec.ModelHub.Filename != nil {
			// Download just one file.
			torrent.Status.Files = []api.FileTracker{
				api.FileTracker{
					Name:  *torrent.Spec.ModelHub.Filename,
					State: api.PendingTrackerState,
				},
			}
		} else {
			// Download the whole repo files.
			repoID := torrent.Spec.ModelHub.ModelID
			// TODO: support URI and ModelScope.
			repo, err := util.ListRepoFiles(repoID)
			if err != nil {
				return ctrl.Result{}, err
			}
			for _, sib := range repo.Siblings {
				torrent.Status.Files = append(torrent.Status.Files,
					api.FileTracker{
						Name:  sib.Rfilename,
						State: api.PendingTrackerState,
					},
				)
			}
		}
		return ctrl.Result{}, r.Client.Status().Update(ctx, torrent)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TorrentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Torrent{}).
		Complete(r)
}
