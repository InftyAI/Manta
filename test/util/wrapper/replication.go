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
	if len(w.Spec.Tuples) == 0 {
		w.Spec.Tuples = append(w.Spec.Tuples, api.Tuple{Source: source})
	} else {
		w.Spec.Tuples[0].Source = source
	}
	return w
}

// Only one tuple be default.
func (w *ReplicationWrapper) DestinationOfAddress(address string) *ReplicationWrapper {
	destination := api.Target{
		URI: &address,
	}
	if len(w.Spec.Tuples) == 0 {
		w.Spec.Tuples = append(w.Spec.Tuples, api.Tuple{Destination: &destination})
	} else {
		w.Spec.Tuples[0].Destination = &destination
	}
	return w
}
