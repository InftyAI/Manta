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
	MaxScore = 100
	MinScore = 0

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
	Revision     string
	NodeSelector map[string]string
}

type NodeInfo struct {
	Name string
}

// Candidate will be used as the result of Filter extension point and input/output of Score extension point.
type Candidate struct {
	// Node represents the candidate nodeTracker.
	Node api.NodeTracker
	// Only set after scoring.
	Score float32
}

// ScoreCandidate will be used after Score extension point for picking best effort nodes.
type ScoreCandidate struct {
	// SourceNodeName represents the the source node name in syncing tasks.
	// It's empty once in downloading tasks.
	SourceNodeName string
	// CandidateNodeName represents the target node name.
	CandidateNodeName string
	// Score for candidate node.
	Score float32
}

// Framework represents the algo about how to pick the candidates among all the peers.
type Framework interface {
	// RegisterPlugins will register the plugins to run.
	RegisterPlugins([]RegisterFunc) error
	// RunFilterPlugins will filter out unsatisfied peers.
	// NodeInfo refers to the source node in syncing tasks, it must not be nil in syncing,
	// on the contrary, it must be nil in downloading tasks.
	RunFilterPlugins(context.Context, ChunkInfo, *NodeInfo, []api.NodeTracker, *cache.Cache) []Candidate
	// RunScorePlugins will calculate the scores of all the peers.
	// NodeInfo refers to the source node in syncing tasks, it must not be nil in syncing,
	// on the contrary, it must be nil in downloading tasks.
	RunScorePlugins(context.Context, ChunkInfo, *NodeInfo, []Candidate, *cache.Cache) []Candidate
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
	// Once NodeInfo is nil, it's a download task, otherwise it's a sync task.
	Filter(context.Context, ChunkInfo, *NodeInfo, api.NodeTracker, *cache.Cache) Status
}

type ScorePlugin interface {
	Plugin
	// Score gets the score of the nodeTracker, it should be ranged between
	// 0 and 100.
	// Once NodeInfo is nil, it's a download task, otherwise it's a sync task.
	Score(context.Context, ChunkInfo, *NodeInfo, api.NodeTracker, *cache.Cache) float32
}
