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

package cache

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	api "github.com/inftyai/manta/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCacheOP(t *testing.T) {
	cache := NewCache()
	chunk1 := api.ChunkTracker{
		ChunkName: "chunk1",
		SizeBytes: 1,
	}
	chunk2 := api.ChunkTracker{
		ChunkName: "chunk2",
		SizeBytes: 2,
	}

	if cache.ChunkExist(chunk1.ChunkName) {
		t.Error("chunk1 should not exist")
	}

	cache.AddChunks([]api.ChunkTracker{chunk1, chunk2}, "node1")

	if !cache.ChunkExist(chunk1.ChunkName) {
		t.Error("chunk1 should exist")
	}

	wantChunks := map[string]*ChunkInfo{
		"chunk1": {
			Name:      "chunk1",
			Nodes:     sets.Set[string](sets.NewString("node1")),
			SizeBytes: 1,
		},
		"chunk2": {
			Name:      "chunk2",
			Nodes:     sets.Set[string](sets.NewString("node1")),
			SizeBytes: 2,
		},
	}

	wantNodes := map[string]sets.Set[string]{
		"node1": sets.Set[string](sets.NewString("chunk1", "chunk2")),
	}

	if diff := cmp.Diff(cache.chunks, wantChunks); diff != "" {
		t.Errorf("unexpected chunks: %v", diff)
	}
	if diff := cmp.Diff(cache.nodes, wantNodes); diff != "" {
		t.Errorf("unexpected nodes: %v", diff)
	}

	newCache := cache.Snapshot()

	if diff := cmp.Diff(cache.chunks, newCache.chunks); diff != "" {
		t.Errorf("unexpected chunks: %v", diff)
	}
	if diff := cmp.Diff(cache.nodes, newCache.nodes); diff != "" {
		t.Errorf("unexpected nodes: %v", diff)
	}

	cache.DeleteChunks([]api.ChunkTracker{chunk1}, "node1")

	wantChunks = map[string]*ChunkInfo{
		"chunk2": {
			Name:      "chunk2",
			Nodes:     sets.Set[string](sets.NewString("node1")),
			SizeBytes: 2,
		},
	}

	wantNodes = map[string]sets.Set[string]{
		"node1": sets.Set[string](sets.NewString("chunk2")),
	}
	if diff := cmp.Diff(cache.chunks, wantChunks); diff != "" {
		t.Errorf("unexpected chunks: %v", diff)
	}
	if diff := cmp.Diff(cache.nodes, wantNodes); diff != "" {
		t.Errorf("unexpected nodes: %v", diff)
	}
}
