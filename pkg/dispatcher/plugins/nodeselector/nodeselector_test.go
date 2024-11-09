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

package nodeselector

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilter(t *testing.T) {
	testCases := []struct {
		name        string
		chunk       framework.ChunkInfo
		nodeTracker api.NodeTracker
		wantStatus  framework.Status
	}{
		{
			name: "nodeSelector matched",
			chunk: framework.ChunkInfo{
				Name:         "chunk1",
				NodeSelector: map[string]string{"zone": "zone1"},
			},
			nodeTracker: api.NodeTracker{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"zone": "zone1"},
				},
			},
			wantStatus: framework.Status{Code: framework.SuccessStatus},
		},
		{
			name: "nodeSelector not match",
			chunk: framework.ChunkInfo{
				Name:         "chunk1",
				NodeSelector: map[string]string{"zone": "zone1"},
			},
			nodeTracker: api.NodeTracker{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"zone": "zone2"},
				},
			},
			wantStatus: framework.Status{Code: framework.UnschedulableStatus},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			plugin, err := New()
			if err != nil {
				t.Errorf("failed to construct plugin: %v", err)
			}

			ns := plugin.(*NodeSelector)

			gotStatus := ns.Filter(ctx, tc.chunk, nil, tc.nodeTracker, nil)
			if diff := cmp.Diff(gotStatus, tc.wantStatus); diff != "" {
				t.Errorf("unexpected status, diff: %v", diff)
			}
		})
	}
}
