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
	"sort"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cons "github.com/inftyai/manta/api"
	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
	"github.com/inftyai/manta/pkg/util"
)

const (
	localhost     = api.URI_LOCALHOST + "://"
	remote        = api.URI_REMOTE + "://"
	workspace     = cons.DefaultWorkspace
	labelHostname = "kubernetes.io/hostname"
)

type Dispatcher struct {
	cache *cache.Cache
	framework.DefaultFramework
}

func NewDispatcher(plugins []framework.RegisterFunc) (*Dispatcher, error) {
	dispatcher := &Dispatcher{
		cache: cache.NewCache(),
	}
	if err := dispatcher.RegisterPlugins(plugins); err != nil {
		return nil, err
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
func (d *Dispatcher) PrepareReplications(ctx context.Context, torrent *api.Torrent, nodeTrackers []api.NodeTracker) (replications []*api.Replication, torrentStatusChanged bool, firstTime bool, err error) {
	if torrent.Status.Repo == nil {
		return nil, false, false, fmt.Errorf("repo is nil, couldn't dispatch chunks")
	}

	// snapshot will deepcopy the cache.
	// Note: because we list the nodeTrackers before so there maybe a bit difference
	// between cache and nodeTrackers.
	cache := d.snapshot()

	pendingNumber := 0
	for i, obj := range torrent.Status.Repo.Objects {
		for j, chunk := range obj.Chunks {
			if chunk.State != api.PendingTrackerState {
				continue
			}

			pendingNumber += 1

			chunk := framework.ChunkInfo{
				Name:         chunk.Name,
				Size:         chunk.SizeBytes,
				Path:         obj.Path,
				Revision:     revision(torrent),
				NodeSelector: torrent.Spec.NodeSelector,
			}

			if d.cache.ChunkExist(chunk.Name) {
				newReplications, err := d.schedulingSyncChunk(ctx, torrent, chunk, nodeTrackers, cache)
				if err != nil {
					return nil, false, false, err
				}
				replications = append(replications, newReplications...)
			} else {
				newReplications, err := d.schedulingDownloadChunk(ctx, torrent, chunk, nodeTrackers, cache)
				if err != nil {
					return nil, false, false, err
				}
				replications = append(replications, newReplications...)
			}

			torrent.Status.Repo.Objects[i].Chunks[j].State = api.ReadyTrackerState
			torrentStatusChanged = true
		}
	}

	// If all the object is pending, it's the first time for dispatching Replications.
	if len(torrent.Status.Repo.Objects) == pendingNumber {
		firstTime = true
	}
	return replications, torrentStatusChanged, firstTime, nil
}

// ReclaimReplications will create replications to delete the chunks.
// This function must be idempotent or we'll create duplicated replications.
func (d *Dispatcher) ReclaimReplications(ctx context.Context, torrent *api.Torrent) (replications []*api.Replication, torrentStatusChanged bool, err error) {
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}
	logger := log.FromContext(ctx)
	logger.Info("start to reclaim chunks", "Torrent", klog.KObj(torrent))

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

func (d *Dispatcher) schedulingDownloadChunk(ctx context.Context, torrent *api.Torrent, chunk framework.ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (replications []*api.Replication, err error) {
	logger := log.FromContext(ctx)
	logger.Info("start to schedule download chunk", "Torrent", klog.KObj(torrent), "chunk", chunk.Name)

	candidates := d.RunFilterPlugins(ctx, chunk, nil, nodeTrackers, cache)

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidate available")
	}

	candidates = d.RunScorePlugins(ctx, chunk, nil, candidates, cache)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > int(*torrent.Spec.Replicas) {
		candidates = candidates[0:int(*torrent.Spec.Replicas)]
	}

	// TODO: once replicas > 1, we only need to download once and sync the rest, will this be better?
	for _, candidate := range candidates {
		replica := buildCreationReplication(torrent, chunk, candidate.Node.Name)
		replications = append(replications, replica)

		// Make sure the snapshotted cache is always updated.
		cache.AddChunks([]api.ChunkTracker{
			{ChunkName: replica.Spec.ChunkName, SizeBytes: replica.Spec.SizeBytes},
		}, candidate.Node.Name)
	}
	return
}

func (d *Dispatcher) schedulingSyncChunk(ctx context.Context, torrent *api.Torrent, chunk framework.ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (replications []*api.Replication, err error) {
	logger := log.FromContext(ctx).WithValues("chunk", chunk.Name)
	logger.Info("start to schedule sync chunk")

	cachedNodeNames := cache.ChunkNodes(chunk.Name)
	replicas := *torrent.Spec.Replicas

	totalCandidates := []framework.ScoreCandidate{}

	// Once the logic becomes complex, we can use a goroutine pool here for concurrency.
	for _, nodeName := range cachedNodeNames {
		nodeInfo := framework.NodeInfo{Name: nodeName}
		// Filter out not qualified nodes.
		candidates := d.RunFilterPlugins(ctx, chunk, &nodeInfo, nodeTrackers, cache)

		if len(candidates) == 0 {
			continue
		}

		// Got the scheduling scores.
		candidates = d.RunScorePlugins(ctx, chunk, &nodeInfo, candidates, cache)

		logger.Info("cached node names", "value", cachedNodeNames)
		for _, candidate := range candidates {
			// Filter out already replicated nodes.
			logger.Info("candidate node name", "value", candidate.Node.Name)
			if util.SetContains(cachedNodeNames, candidate.Node.Name) {
				replicas -= 1
				continue
			}

			totalCandidates = append(totalCandidates, framework.ScoreCandidate{SourceNodeName: nodeName, CandidateNodeName: candidate.Node.Name, Score: candidate.Score})
		}
	}

	// We have enough replicated nodes.
	if replicas <= 0 {
		logger.V(1).Info("Have enough replicas, no need to sync anymore")
		return nil, nil
	}

	if len(totalCandidates) == 0 {
		return nil, fmt.Errorf("no candidate available")
	}

	if len(totalCandidates) > int(replicas) {
		sort.Slice(totalCandidates, func(i, j int) bool {
			return totalCandidates[i].Score > totalCandidates[j].Score
		})

		totalCandidates = totalCandidates[0:replicas]
	}

	for _, candidate := range totalCandidates {
		replica := buildSyncReplication(torrent, chunk, candidate.SourceNodeName, candidate.CandidateNodeName)
		replications = append(replications, replica)

		// Make sure the snapshot cache is always updated.
		cache.AddChunks([]api.ChunkTracker{
			{ChunkName: replica.Spec.ChunkName, SizeBytes: replica.Spec.SizeBytes},
		}, candidate.CandidateNodeName)
	}

	return replications, nil
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
	generatedName := util.GenerateName(nodeName)
	name := chunk.Name + "--" + generatedName

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
				URI: ptr.To[string](localhost + workspace + repoName + "/blobs/" + chunk.Name),
			},
			SizeBytes: chunk.Size,
		},
	}
}

func buildSyncReplication(torrent *api.Torrent, chunk framework.ChunkInfo, sourceName string, targetName string) *api.Replication {
	repoName := hubRepoName(torrent.Spec.Hub)
	generatedName := util.GenerateName(targetName)
	name := chunk.Name + "--" + generatedName

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
			NodeName:  targetName,
			ChunkName: chunk.Name,
			Source: api.Target{
				URI: ptr.To[string](remote + sourceName + "@" + workspace + repoName + "/blobs/" + chunk.Name),
			},
			Destination: &api.Target{
				URI: ptr.To[string](localhost + workspace + repoName + "/snapshots/" + chunk.Revision + "/" + chunk.Path),
			},
			SizeBytes: chunk.Size,
		},
	}
}

func buildDeletionReplication(torrent *api.Torrent, chunk framework.ChunkInfo, nodeName string) *api.Replication {
	repoName := hubRepoName(torrent.Spec.Hub)
	generatedName := util.GenerateName(nodeName)
	name := chunk.Name + "--" + generatedName + "--" + "d"

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
				URI: ptr.To[string](localhost + workspace + repoName + "/snapshots/" + chunk.Revision + "/" + chunk.Path),
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
