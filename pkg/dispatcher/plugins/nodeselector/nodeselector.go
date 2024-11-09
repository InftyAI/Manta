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

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
)

var _ framework.FilterPlugin = &NodeSelector{}

type NodeSelector struct{}

func New() (framework.Plugin, error) {
	return &NodeSelector{}, nil
}

func (ns *NodeSelector) Name() string {
	return "NodeSelector"
}

func (ns *NodeSelector) Filter(ctx context.Context, chunkInfo framework.ChunkInfo, _ *framework.NodeInfo, nodeTracker api.NodeTracker, cache *cache.Cache) framework.Status {
	// In a big cluster, this is serious maybe we should have a preFilter extension point.
	for k, v := range chunkInfo.NodeSelector {
		value, ok := nodeTracker.Labels[k]
		if !ok || value != v {
			return framework.Status{Code: framework.UnschedulableStatus}
		}
	}

	return framework.Status{Code: framework.SuccessStatus}
}
