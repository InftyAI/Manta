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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			if chunk.State == api.PendingTrackerState {
				replication := &api.Replication{
					TypeMeta: v1.TypeMeta{
						Kind:       "Replication",
						APIVersion: api.GroupVersion.String(),
					},
					ObjectMeta: v1.ObjectMeta{
						Name: chunk.Name,
					},
					Spec: api.ReplicationSpec{
						Tuples: []api.Tuple{
							{
								// TODO: source could be local or remote
								Source: api.Target{
									ChunkName: chunk.Name,
								},
								Destination: &api.Target{
									ChunkName: chunk.Name,
								},
							},
						},
					},
				}

				replications = append(replications, replication)

				// Update the chunk state as well, we'll update the status later.
				chunk.State = api.DownloadTrackerState
				torrentStatusChanged = true
			}
		}
	}
	return replications, torrentStatusChanged, nil
}
