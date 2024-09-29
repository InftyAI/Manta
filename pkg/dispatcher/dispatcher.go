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
	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ Framework = &DefaultDownloader{}
var _ Framework = &DefaultSyncer{}

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
func (d *Dispatcher) PrepareReplications(torrent *api.Torrent) ([]*api.Replication, bool, error) {
	// Make sure this will not happen, just in case of panic.
	if torrent.Status.Repo == nil {
		return nil, false, nil
	}

	replications := []*api.Replication{}
	var torrentStatusChanged bool

	for _, obj := range torrent.Status.Repo.Objects {
		for _, chunk := range obj.Chunks {
			// TODO: we should also compare the desired Replicas with the real Replicas here.
			if chunk.State == api.PendingTrackerState {

				// Create a Replication for each spec.replicas.
				for i := 0; i < int(*torrent.Spec.Replicas); i++ {
					replica, err := buildReplication(torrent, obj.Path, chunk.Name)
					if err != nil {
						return nil, false, err
					}
					replications = append(replications, replica)
				}
				// Update the chunk state as well, we'll update the status later.
				chunk.State = api.DownloadTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}

func buildReplication(torrent *api.Torrent, objPath string, chunkName string) (*api.Replication, error) {
	name, err := util.GenerateName(chunkName)
	if err != nil {
		return nil, err
	}

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
			NodeName:   "unknown",
			RepoName:   torrent.Status.Repo.Name,
			ObjectPath: objPath,
			ChunkName:  chunkName,
			Tuples: []api.Tuple{
				{
					// TODO: source could be local or remote
					Source: api.Target{
						Address: ptr.To[string]("unknown"),
					},
					Destination: &api.Target{
						Address: ptr.To[string]("unknown"),
					},
				},
			},
		},
	}, nil
}
