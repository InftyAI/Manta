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

package dispatcher

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/inftyai/manta/agent/pkg/util"
	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
)

const (
	localHost = api.URI_LOCALHOST + "://"
	workspace = util.DefaultWorkspace
)

// DefaultDownloader helps to download the chunks.
type DefaultDownloader struct {
	framework.DefaultFramework
}

func newDefaultDownloader(plugins []framework.RegisterFunc) (*DefaultDownloader, error) {
	downloader := &DefaultDownloader{}
	if err := downloader.RegisterPlugins(plugins); err != nil {
		return nil, err
	}
	return downloader, nil
}

// DefaultSyncer helps to sync the chunks in the p2p network.
type DefaultSyncer struct {
	framework.DefaultFramework
}

func newDefaultSyncer(plugins []framework.RegisterFunc) (*DefaultSyncer, error) {
	syncer := &DefaultSyncer{}
	if err := syncer.RegisterPlugins(plugins); err != nil {
		return nil, err
	}
	return syncer, nil
}

type Dispatcher struct {
	cache      *cache.Cache
	downloader *DefaultDownloader
	syncer     *DefaultSyncer
}

func NewDispatcher(downloadPlugins []framework.RegisterFunc, syncPlugins []framework.RegisterFunc) (*Dispatcher, error) {
	downloader, err := newDefaultDownloader(downloadPlugins)
	if err != nil {
		return nil, err
	}
	syncer, err := newDefaultSyncer(syncPlugins)
	if err != nil {
		return nil, err
	}

	dispatcher := &Dispatcher{
		downloader: downloader,
		syncer:     syncer,
		cache:      cache.NewCache(),
	}

	return dispatcher, nil
}

func (d *Dispatcher) snapshot() *cache.Cache {
	return d.cache.Snapshot()
}

// PrepareReplications will construct the replications needed to created and
// update the torrent status the same time.
// This function must be idempotent or we'll create duplicated replications.
// Note: make sure the same download/sync task will not be sent to the same node,
// or we have to introduce file lock when downloading chunks.
func (d *Dispatcher) PrepareReplications(ctx context.Context, torrent *api.Torrent, nodeTrackers []api.NodeTracker) (replications []*api.Replication, torrentStatusChanged bool, err error) {
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}

	logger := log.FromContext(ctx)

	// snapshot will deepcopy the cache.
	// Note: because we list the nodeTrackers before so there maybe a bit difference
	// between cache and nodeTrackers.
	cache := d.snapshot()

	for i, obj := range torrent.Status.Repo.Objects {
		for j, chunk := range obj.Chunks {
			if chunk.State == api.PendingTrackerState {

				chunk := framework.ChunkInfo{
					Name:         chunk.Name,
					Size:         chunk.SizeBytes,
					Path:         obj.Path,
					Revision:     revision(torrent),
					NodeSelector: torrent.Spec.NodeSelector,
				}

				if d.cache.ChunkExist(chunk.Name) {
					replications, err = d.schedulingSyncChunk(ctx, torrent, chunk, nodeTrackers, cache)
					if err != nil {
						return nil, false, err
					}
				} else {
					newReplications, err := d.schedulingDownloadChunk(ctx, torrent, chunk, nodeTrackers, cache)
					if err != nil {
						// Once err, ignore for this dispatching cycle, will retry for next cycle.
						logger.Error(err, "failed to dispatch chunk for downloading", "chunk", chunk.Name)
						continue
					}
					replications = append(replications, newReplications...)
				}

				torrent.Status.Repo.Objects[i].Chunks[j].State = api.ReadyTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}

// ReclaimReplications will create replications to delete the chunks.
// This function must be idempotent or we'll create duplicated replications.
func (d *Dispatcher) ReclaimReplications(ctx context.Context, torrent *api.Torrent) (replications []*api.Replication, torrentStatusChanged bool, err error) {
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}
	logger := log.FromContext(ctx)

	for i, obj := range torrent.Status.Repo.Objects {
		for j, chunk := range obj.Chunks {
			if chunk.State != api.DeletingTrackerState {
				nodeNames := d.cache.ChunkNodes(chunk.Name)
				logger.Info("reclaiming replications", "chunk", chunk.Name, "nodes", nodeNames)
				for _, nodeName := range nodeNames {
					chunkInfo := framework.ChunkInfo{
						Name:     chunk.Name,
						Path:     obj.Path,
						Revision: revision(torrent),
						Size:     0,
					}
					replication := buildDeletionReplication(torrent, chunkInfo, nodeName)
					replications = append(replications, replication)
				}
				torrent.Status.Repo.Objects[i].Chunks[j].State = api.DeletingTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}

func (d *Dispatcher) schedulingSyncChunk(ctx context.Context, torrent *api.Torrent, chunk framework.ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (replications []*api.Replication, err error) {
	candidates := d.syncer.RunFilterPlugins(ctx, chunk, nodeTrackers, cache)

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available candidate")
	}

	candidates = d.downloader.RunScorePlugins(ctx, chunk, candidates, *torrent.Spec.Replicas, cache)

	// TODO: we only need to download once and sync the rest, will this be better?
	// Maybe file size is a big take we should consider.
	for _, candidate := range candidates {
		replica := buildCreationReplication(torrent, chunk, candidate.Name)
		replications = append(replications, replica)

		// Make sure the snapshotted cache is always updated.
		cache.AddChunks([]api.ChunkTracker{
			{ChunkName: replica.Spec.ChunkName, SizeBytes: replica.Spec.SizeBytes},
		}, candidate.Name)
	}
	return
}

func (d *Dispatcher) schedulingDownloadChunk(ctx context.Context, torrent *api.Torrent, chunk framework.ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (replications []*api.Replication, err error) {
	candidates := d.downloader.RunFilterPlugins(ctx, chunk, nodeTrackers, cache)

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available candidate")
	}

	candidates = d.downloader.RunScorePlugins(ctx, chunk, candidates, *torrent.Spec.Replicas, cache)

	// TODO: we only need to download once and sync the rest, will this be better?
	// Maybe file size is a big take we should consider.
	for _, candidate := range candidates {
		replica := buildCreationReplication(torrent, chunk, candidate.Name)
		replications = append(replications, replica)

		// Make sure the snapshotted cache is always updated.
		cache.AddChunks([]api.ChunkTracker{
			{ChunkName: replica.Spec.ChunkName, SizeBytes: replica.Spec.SizeBytes},
		}, candidate.Name)
	}
	return
}

func (d *Dispatcher) UpdateNodeTracker(old *api.NodeTracker, new *api.NodeTracker) {
	// Batch OPs to avoid lock races.
	toDelete, toAdd := chunksDiff(old.Spec.Chunks, new.Spec.Chunks)
	d.cache.DeleteChunks(toDelete, new.Name)
	d.cache.AddChunks(toAdd, new.Name)
}

func (d *Dispatcher) AddNodeTracker(obj *api.NodeTracker) {
	d.cache.AddChunks(obj.Spec.Chunks, obj.Name)
}

func (d *Dispatcher) DeleteNodeTracker(obj *api.NodeTracker) {
	d.cache.DeleteChunks(obj.Spec.Chunks, obj.Name)
}

// toDelete includes chunk in old but not in new,
// toAdd includes chunk in new but not in old.
func chunksDiff(old []api.ChunkTracker, new []api.ChunkTracker) (toDelete []api.ChunkTracker, toAdd []api.ChunkTracker) {
	for _, c := range old {
		if !chunkIn(new, c) {
			toDelete = append(toDelete, c)
		}
	}

	for _, c := range new {
		if !chunkIn(old, c) {
			toAdd = append(toAdd, c)
		}
	}
	return
}

func chunkIn(chunks []api.ChunkTracker, chunk api.ChunkTracker) bool {
	for _, c := range chunks {
		if c.ChunkName == chunk.ChunkName {
			return true
		}
	}
	return false
}

func buildCreationReplication(torrent *api.Torrent, chunk framework.ChunkInfo, nodeName string) *api.Replication {
	repoName := hubRepoName(torrent.Spec.Hub)
	name := torrent.Name + "--" + chunk.Name + "--" + nodeName

	return &api.Replication{
		TypeMeta: v1.TypeMeta{
			Kind:       "Replication",
			APIVersion: api.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			OwnerReferences: []v1.OwnerReference{
				{
					Kind:               "Torrent",
					APIVersion:         api.GroupVersion.String(),
					Name:               torrent.Name,
					UID:                torrent.UID,
					BlockOwnerDeletion: ptr.To(true),
					Controller:         ptr.To(true),
				},
			},
			Labels: map[string]string{
				api.TorrentNameLabelKey: torrent.Name,
			},
		},
		Spec: api.ReplicationSpec{
			NodeName:  nodeName,
			ChunkName: chunk.Name,
			Source: api.Target{
				// TODO: once we support loading files from s3, we should change the logic here.
				Hub: &api.Hub{
					Name:   torrent.Spec.Hub.Name,
					RepoID: torrent.Spec.Hub.RepoID,
					// TODO: support multiple chunks for one file in the future.
					Filename: &chunk.Path,
					Revision: torrent.Spec.Hub.Revision,
				},
			},
			Destination: &api.Target{
				URI: ptr.To[string](localHost + workspace + repoName + "/blobs/" + chunk.Name),
			},
			SizeBytes: chunk.Size,
		},
	}
}

func buildDeletionReplication(torrent *api.Torrent, chunk framework.ChunkInfo, nodeName string) *api.Replication {
	// TODO: once we support URI, we may change the logic here as well.
	repoName := hubRepoName(torrent.Spec.Hub)
	// Add the nodeName to make sure the replication name is unique because we may
	// create several replications to remove the same chunk from different nodes.
	name := torrent.Name + "--" + chunk.Name + "--" + nodeName

	return &api.Replication{
		TypeMeta: v1.TypeMeta{
			Kind:       "Replication",
			APIVersion: api.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			OwnerReferences: []v1.OwnerReference{
				{
					Kind:               "Torrent",
					APIVersion:         api.GroupVersion.String(),
					Name:               torrent.Name,
					UID:                torrent.UID,
					BlockOwnerDeletion: ptr.To(true),
					Controller:         ptr.To(true),
				},
			},
			Labels: map[string]string{
				api.TorrentNameLabelKey: torrent.Name,
			},
		},
		Spec: api.ReplicationSpec{
			NodeName:  nodeName,
			ChunkName: chunk.Name,
			Source: api.Target{
				URI: ptr.To[string](localHost + workspace + repoName + "/snapshots/" + chunk.Revision + "/" + chunk.Path),
			},
			Destination: nil,
			SizeBytes:   chunk.Size,
		},
	}
}

func hubRepoName(hub *api.Hub) string {
	return strings.ReplaceAll(hub.RepoID, "/", "--")
}

func revision(torrent *api.Torrent) string {
	if torrent.Spec.Hub != nil {
		return *torrent.Spec.Hub.Revision
	}
	// Default to "main" for URI
	return "main"
}
