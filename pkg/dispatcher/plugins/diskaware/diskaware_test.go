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

package diskaware

import (
	"context"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
	"github.com/inftyai/manta/test/util/wrapper"
)

func TestFilter(t *testing.T) {
	testCases := []struct {
		name        string
		chunk       framework.ChunkInfo
		nodeTracker api.NodeTracker
		cache       func() *cache.Cache
		wantStatus  framework.Status
	}{
		{
			name: "small chunk size with empty cache",
			chunk: framework.ChunkInfo{
				Name: "chunk1",
				Size: 512,
			},
			nodeTracker: *wrapper.MakeNodeTracker("node1").SizeLimit("10Mi").Obj(),
			cache:       func() *cache.Cache { return cache.NewCache().Snapshot() },
			wantStatus:  framework.Status{Code: framework.SuccessStatus},
		},
		{
			name: "small chunk size with not empty cache",
			chunk: framework.ChunkInfo{
				Name: "chunk1",
				Size: 512,
			},
			nodeTracker: *wrapper.MakeNodeTracker("node1").SizeLimit("10Mi").Obj(),
			cache: func() *cache.Cache {
				c := cache.NewCache()
				c.AddChunks([]api.ChunkTracker{
					{
						ChunkName: "chunk1",
						SizeBytes: 1 * 1024 * 1024, // 1Mi
					},
				}, "node1")

				return c.Snapshot()
			},
			wantStatus: framework.Status{Code: framework.SuccessStatus},
		},
		{
			name: "big chunk size with cache",
			chunk: framework.ChunkInfo{
				Name: "chunk1",
				Size: 9 * 1024 * 1024,
			},
			nodeTracker: *wrapper.MakeNodeTracker("node1").SizeLimit("10Mi").Obj(),
			cache: func() *cache.Cache {
				c := cache.NewCache()
				c.AddChunks([]api.ChunkTracker{
					{
						ChunkName: "chunk1",
						SizeBytes: 1*1024*1024 + 1, // 1Mi + 1 Bytes
					},
				}, "node1")

				return c.Snapshot()
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

			ns := plugin.(*DiskAware)

			gotStatus := ns.Filter(ctx, tc.chunk, nil, tc.nodeTracker, tc.cache())
			if diff := cmp.Diff(gotStatus, tc.wantStatus); diff != "" {
				t.Errorf("unexpected status, diff: %v", diff)
			}
		})
	}
}

func TestScore(t *testing.T) {
	testCases := []struct {
		name        string
		chunk       framework.ChunkInfo
		nodeTracker api.NodeTracker
		cache       func() *cache.Cache
		wantScore   float32
	}{
		{
			name: "empty cache",
			chunk: framework.ChunkInfo{
				Name: "chunk1",
				Size: 512,
			},
			nodeTracker: *wrapper.MakeNodeTracker("node1").SizeLimit("2Mi").Obj(),
			cache:       func() *cache.Cache { return cache.NewCache().Snapshot() },
			wantScore:   99.98,
		},
		{
			name: "non empty cache",
			chunk: framework.ChunkInfo{
				Name: "chunk1",
				Size: 512,
			},
			nodeTracker: *wrapper.MakeNodeTracker("node1").SizeLimit("2Mi").Obj(),
			cache: func() *cache.Cache {
				c := cache.NewCache()
				c.AddChunks([]api.ChunkTracker{
					{
						ChunkName: "chunk1",
						SizeBytes: 1 * 1024 * 1024, // 1Mi
					},
				}, "node1")

				return c.Snapshot()
			},
			wantScore: 49.98,
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

			ns := plugin.(*DiskAware)

			gotScore := ns.Score(ctx, tc.chunk, nil, tc.nodeTracker, tc.cache())
			if math.Abs(float64(gotScore-tc.wantScore)) > 0.01 {
				t.Errorf("unexpected score, want %v, got %v", tc.wantScore, gotScore)
			}
		})
	}
}
