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

package framework

import (
	"context"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
)

type Status struct {
	Code StatusCode
}

type StatusCode string

const (
	SuccessStatus       StatusCode = "success"
	UnschedulableStatus StatusCode = "Unschedulable"
)

// Download represents the methods to download a chunk.
type Download interface {
	Framework
}

// Sync represents the methods to sync a chunk.
type Sync interface {
	Framework
}

type ChunkInfo struct {
	Name         string
	Size         int64
	Path         string
	NodeSelector map[string]string
}

// Framework represents the algo about how to pick the candidates among all the peers.
type Framework interface {
	// RegisterPlugins will register the plugins to run.
	RegisterPlugins([]RegisterFunc) error
	// RunFilterPlugins will filter out unsatisfied peers.
	RunFilterPlugins(context.Context, ChunkInfo, []api.NodeTracker, *cache.Cache) []api.NodeTracker
	// RunScorePlugins will calculate the scores of all the peers.
	RunScorePlugins(context.Context, ChunkInfo, []api.NodeTracker, int32, *cache.Cache) []api.NodeTracker
}

// Plugin is the parent type for all the framework plugins.
// the same time.
type Plugin interface {
	Name() string
}

type PreFilterPlugin interface {
	Plugin
	// PreFilter helps to do some computing before calling Filter plugins.
	PreFilter(context.Context, ChunkInfo, *cache.Cache) Status
}

type FilterPlugin interface {
	Plugin
	// Filter helps to filter out unrelated nodes.
	Filter(context.Context, ChunkInfo, api.NodeTracker, *cache.Cache) Status
}

type ScorePlugin interface {
	Plugin
	// Score gets the score of the nodeTracker, it should be ranged between
	// 0 and 100.
	Score(context.Context, ChunkInfo, api.NodeTracker, *cache.Cache) float32
}
