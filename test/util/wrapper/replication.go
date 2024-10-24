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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/inftyai/manta/api/v1alpha1"
)

type ReplicationWrapper struct {
	api.Replication
}

func MakeReplication(name string) *ReplicationWrapper {
	return &ReplicationWrapper{
		api.Replication{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func (w *ReplicationWrapper) Obj() *api.Replication {
	return &w.Replication
}

func (w *ReplicationWrapper) NodeName(name string) *ReplicationWrapper {
	w.Spec.NodeName = name
	return w
}

func (w *ReplicationWrapper) ChunkName(name string) *ReplicationWrapper {
	w.Spec.ChunkName = name
	return w
}

func (w *ReplicationWrapper) SizeBytes(size int64) *ReplicationWrapper {
	w.Spec.SizeBytes = size
	return w
}

// Only one tuple be default.
func (w *ReplicationWrapper) SourceOfModelHub(name, modelID, revision, filename string) *ReplicationWrapper {
	source := api.Target{
		ModelHub: &api.ModelHub{
			ModelID: modelID,
		},
	}
	if name != "" {
		source.ModelHub.Name = &name
	}
	if revision != "" {
		source.ModelHub.Revision = &revision
	}
	if filename != "" {
		source.ModelHub.Filename = &filename
	}

	w.Spec.Source = source
	return w
}

// Only one tuple be default.
func (w *ReplicationWrapper) DestinationOfAddress(address string) *ReplicationWrapper {
	destination := api.Target{
		URI: &address,
	}
	w.Spec.Destination = &destination
	return w
}
