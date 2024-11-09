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
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ Framework = &DefaultFramework{}

type DefaultFramework struct {
	registry Registry
}

func (df *DefaultFramework) RegisterPlugins(fns []RegisterFunc) error {
	if df.registry == nil {
		df.registry = make(Registry)
	}

	for _, fn := range fns {
		if err := df.registry.Register(fn); err != nil {
			return err
		}
	}

	return nil
}

func (df *DefaultFramework) RunFilterPlugins(ctx context.Context, chunk ChunkInfo, nodeInfo *NodeInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (candidates []Candidate) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var status Status

	logger := log.FromContext(ctx)

	// TODO: consider performance issue once thousands of nodeTrackers in the cluster.
	for _, nt := range nodeTrackers {
		for _, plugin := range df.registry {
			if p, ok := plugin.(FilterPlugin); ok {
				status = p.Filter(ctx, chunk, nodeInfo, nt, cache)
				if status.Code != SuccessStatus {
					logger.Info("filter out plugin", "plugin", plugin.Name(), "node", nt.Name, "file", chunk.Path, "chunk", chunk.Name)
					break
				}
			}
		}
		if status.Code == SuccessStatus {
			candidates = append(candidates, Candidate{Node: nt})
		}
	}

	return candidates
}
func (df *DefaultFramework) RunScorePlugins(ctx context.Context, chunk ChunkInfo, nodeInfo *NodeInfo, candidates []Candidate, cache *cache.Cache) []Candidate {
	logger := log.FromContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, nt := range candidates {
		var totalScore float32

		for _, plugin := range df.registry {
			if p, ok := plugin.(ScorePlugin); ok {
				score := p.Score(ctx, chunk, nodeInfo, nt.Node, cache)

				logger.V(10).Info("calculate plugin score", "plugin", plugin.Name(), "node", nt.Node.Name, "file", chunk.Path, "chunk", chunk.Name, "score", score)
				totalScore += standardScore(score)
			}
		}
		candidates[i].Score = totalScore
	}

	return candidates
}

func standardScore(score float32) float32 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
