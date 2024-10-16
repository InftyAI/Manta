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
	"errors"
	"strconv"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	api "github.com/inftyai/manta/api/v1alpha1"
)

var _ Framework = &DefaultDownloader{}
var _ Framework = &DefaultSyncer{}

const (
	localHost = "localhost://"
)

type DefaultDownloader struct {
	plugins []string
}

func (dd *DefaultDownloader) RegisterPlugins(plugins []string) {
	dd.plugins = plugins
}

func (d *DefaultDownloader) RunFilterPlugins() {
}

func (d *DefaultDownloader) RunScorePlugins() {

}

type DefaultSyncer struct {
	plugins []string
}

func (ds *DefaultSyncer) RegisterPlugins(plugins []string) {
	ds.plugins = plugins
}

func (ds *DefaultSyncer) RunFilterPlugins() {
}

func (ds *DefaultSyncer) RunScorePlugins() {

}

type Dispatcher struct {
	cache      *cache
	downloader *DefaultDownloader
	syncer     *DefaultSyncer
}

func NewDispatcher(downloadPlugins []string, syncPlugins []string) *Dispatcher {
	downloader := &DefaultDownloader{}
	downloader.RegisterPlugins(downloadPlugins)
	syncer := &DefaultSyncer{}
	syncer.RegisterPlugins(syncPlugins)

	dispatcher := &Dispatcher{
		downloader: downloader,
		syncer:     syncer,
		cache:      &cache{},
	}

	return dispatcher
}

// PrepareReplications will construct the replications needed to created and
// update the torrent status the same time.
// Note: make sure the same download/sync task will not be sent to the same node,
// or we have to introduce file lock when downloading chunks.
func (d *Dispatcher) PrepareReplications(torrent *api.Torrent) ([]*api.Replication, bool, error) {
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}

	replications := []*api.Replication{}
	var torrentStatusChanged bool

	for i, obj := range torrent.Status.Repo.Objects {
		for j, chunk := range obj.Chunks {
			// TODO: we should also compare the desired Replicas with the real Replicas here.
			if chunk.State == api.PendingTrackerState {

				// Create a Replication for each spec.replicas.
				for i := 0; i < int(*torrent.Spec.Replicas); i++ {
					replica, err := buildReplication(torrent, obj.Path, chunk.Name, chunk.SizeBytes, i)
					if err != nil {
						return nil, false, err
					}
					replications = append(replications, replica)
				}
				// Update the chunk state as well, we'll update the status later, next time, we'll not
				// construct the Replication anymore.
				torrent.Status.Repo.Objects[i].Chunks[j].State = api.TrackedTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}

func buildReplication(torrent *api.Torrent, objPath string, chunkName string, size int64, index int) (*api.Replication, error) {
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
			Name: chunkName + "--" + strconv.Itoa(index),
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
				api.ChunkNameLabelKey:   chunkName,
			},
		},
		Spec: api.ReplicationSpec{
			// TODO:
			NodeName: "unknown",
			Tuples: []api.Tuple{
				{
					Source: api.Target{
						ModelHub: &api.ModelHub{
							Name:    torrent.Spec.ModelHub.Name,
							ModelID: torrent.Spec.ModelHub.ModelID,
							// TODO: support multiple chunks for one file in the future.
							Filename: &objPath,
							Revision: torrent.Spec.ModelHub.Revision,
						},
					},
					Destination: &api.Target{
						URI: ptr.To[string](localHost + api.DefaultWorkspace + repoName + "/blobs/" + chunkName),
					},
					SizeBytes: size,
				},
			},
		},
	}, nil
}

func repoName(modelHub *api.ModelHub) string {
	return strings.ReplaceAll(modelHub.ModelID, "/", "--")
}
