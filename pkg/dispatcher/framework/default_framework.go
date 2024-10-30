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
	"github.com/inftyai/manta/pkg/util"
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

func (df *DefaultFramework) RunFilterPlugins(ctx context.Context, chunk ChunkInfo, nodeTrackers []api.NodeTracker, cache *cache.Cache) (candidates []api.NodeTracker) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var status Status

	logger := log.FromContext(ctx)

	// TODO: consider performance issue once thousands of nodeTrackers in the cluster.
	for _, nt := range nodeTrackers {
		for _, plugin := range df.registry {
			if p, ok := plugin.(FilterPlugin); ok {
				status = p.Filter(ctx, chunk, nt, cache)
				if status.Code != SuccessStatus {
					logger.Info("filter out plugin", "plugin", plugin.Name(), "node", nt.Name, "file", chunk.Path, "chunk", chunk.Name)
					break
				}
			}
		}
		if status.Code == SuccessStatus {
			candidates = append(candidates, nt)
		}
	}

	return candidates
}

func (df *DefaultFramework) RunScorePlugins(ctx context.Context, chunk ChunkInfo, nodeTrackers []api.NodeTracker, replicas int32, cache *cache.Cache) (candidates []api.NodeTracker) {
	logger := log.FromContext(ctx)

	if len(nodeTrackers) <= int(replicas) {
		logger.Info("return all candidates", "file", chunk.Path, "chunk", chunk.Name)
		return nodeTrackers
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	scores := make([]float32, len(nodeTrackers))
	for i, nt := range nodeTrackers {
		var totalScore float32

		for _, plugin := range df.registry {
			if p, ok := plugin.(ScorePlugin); ok {
				score := p.Score(ctx, chunk, nt, cache)

				logger.Info("calculate plugin score", "plugin", plugin.Name(), "node", nt.Name, "file", chunk.Path, "chunk", chunk.Name, "score", score)
				totalScore += standardScore(score)
			}
		}
		scores[i] = totalScore
	}

	indices := util.TopNIndices(scores, int(replicas))

	for _, index := range indices {
		candidates = append(candidates, nodeTrackers[index])
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
