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

package wrapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/inftyai/manta/api/v1alpha1"
)

type TorrentWrapper struct {
	api.Torrent
}

func MakeTorrent(name string) *TorrentWrapper {
	return &TorrentWrapper{
		api.Torrent{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func (w *TorrentWrapper) Obj() *api.Torrent {
	return &w.Torrent
}

func (w *TorrentWrapper) Hub(name string, repoID string, filename string) *TorrentWrapper {
	if w.Spec.Hub == nil {
		w.Spec.Hub = &api.Hub{}
	}
	if name != "" {
		w.Spec.Hub.Name = &name
	}
	w.Spec.Hub.RepoID = repoID
	if filename != "" {
		w.Spec.Hub.Filename = &filename
	}
	return w
}

func (w *TorrentWrapper) Replicas(replicas int32) *TorrentWrapper {
	w.Spec.Replicas = &replicas
	return w
}

func (w *TorrentWrapper) ReclaimPolicy(policy api.ReclaimPolicy) *TorrentWrapper {
	w.Spec.ReclaimPolicy = &policy
	return w
}

func (w *TorrentWrapper) NodeSelector(k, v string) *TorrentWrapper {
	if w.Spec.NodeSelector == nil {
		w.Spec.NodeSelector = map[string]string{}
	}
	w.Spec.NodeSelector[k] = v
	return w
}

func (w *TorrentWrapper) Preheat(yesOrNo bool) *TorrentWrapper {
	w.Spec.Preheat = &yesOrNo
	return w
}

func (w *TorrentWrapper) TTL(number int32) *TorrentWrapper {
	ttl := time.Duration(number) * time.Second
	w.Spec.TTLSecondsAfterReady = &ttl
	return w
}
