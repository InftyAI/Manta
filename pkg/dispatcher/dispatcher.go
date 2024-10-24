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
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
)

const (
	localHost        = "localhost://"
	defaultWorkspace = "/workspace/models/"
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
// Note: make sure the same download/sync task will not be sent to the same node,
// or we have to introduce file lock when downloading chunks.
func (d *Dispatcher) PrepareReplications(ctx context.Context, torrent *api.Torrent, nodeTrackers []api.NodeTracker) (replications []*api.Replication, torrentStatusChanged bool, err error) {
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}

	// snapshot will deepcopy the cache.
	// Note: because we list the nodeTrackers before so there maybe a bit difference
	// between cache and nodeTrackers.
	cache := d.snapshot()

	for i, obj := range torrent.Status.Repo.Objects {
		for j, chunk := range obj.Chunks {
			if chunk.State == api.PendingTrackerState {
				if _, ok := d.cache.ChunkExist(chunk.Name); ok {
					// Sync chunks here.
					replications, err = d.syncChunk()
					if err != nil {
						return nil, false, err
					}
				} else {
					// Download chunks here.
					chunk := framework.ChunkInfo{
						Name:         chunk.Name,
						Size:         chunk.SizeBytes,
						Path:         obj.Path,
						NodeSelector: torrent.Spec.NodeSelector,
					}

					newReplications, err := d.downloadChunk(ctx, torrent, chunk, nodeTrackers, cache)
					if err != nil {
						return nil, false, err
					}
					replications = append(replications, newReplications...)
				}

				torrent.Status.Repo.Objects[i].Chunks[j].State = api.TrackedTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}

func (d *Dispatcher) CleanupReplications(ctx context.Context, torrent *api.Torrent) (err error) {
	if torrent.Status.Repo == nil {
		return nil
	}
	return nil
}

func (d *Dispatcher) syncChunk() (replications []*api.Replication, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Dispatcher) downloadChunk(ctx context.Context, torrent *api.Torrent, chunk framework.ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (replications []*api.Replication, err error) {
	candidates := d.downloader.RunFilterPlugins(ctx, chunk, nodeTrackers, cache)

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available candidate")
	}

	candidates = d.downloader.RunScorePlugins(ctx, chunk, candidates, *torrent.Spec.Replicas, cache)

	// TODO: we only need to download once and sync the rest, will this be better?
	// Maybe file size is a big take we should consider.
	for i, candidate := range candidates {
		replica, err := buildReplication(torrent, chunk, i, candidate.Name)
		if err != nil {
			return nil, err
		}

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

func buildReplication(torrent *api.Torrent, chunk framework.ChunkInfo, index int, nodeName string) (*api.Replication, error) {
	// Support modelHub only right now.
	if torrent.Spec.ModelHub == nil {
		return nil, errors.New("unimplemented")
	}

	repoName := repoName(torrent.Spec.ModelHub)

	return &api.Replication{
		TypeMeta: v1.TypeMeta{
			Kind:       "Replication",
			APIVersion: api.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: torrent.Name + "--" + chunk.Name + "--" + strconv.Itoa(index),
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
				ModelHub: &api.ModelHub{
					Name:    torrent.Spec.ModelHub.Name,
					ModelID: torrent.Spec.ModelHub.ModelID,
					// TODO: support multiple chunks for one file in the future.
					Filename: &chunk.Path,
					Revision: torrent.Spec.ModelHub.Revision,
				},
			},
			Destination: &api.Target{
				URI: ptr.To[string](localHost + defaultWorkspace + repoName + "/blobs/" + chunk.Name),
			},
			SizeBytes: chunk.Size,
		},
	}, nil
}

func repoName(modelHub *api.ModelHub) string {
	return strings.ReplaceAll(modelHub.ModelID, "/", "--")
}
