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

package gnumber

import (
	"context"
	"runtime"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
)

var _ framework.FilterPlugin = &GNumber{}
var _ framework.ScorePlugin = &GNumber{}

const (
	defaultGoroutineLimit = 1000
)

type GNumber struct{}

func New() (framework.Plugin, error) {
	return &GNumber{}, nil
}

func (g *GNumber) Name() string {
	return "GNumber"
}

func (g *GNumber) Filter(ctx context.Context, chunk framework.ChunkInfo, nodeTracker api.NodeTracker, cache *cache.Cache) framework.Status {
	if cache.ChunkExistInNode(nodeTracker.Name, chunk.Name) {
		return framework.Status{Code: framework.SuccessStatus}
	}
	return framework.Status{Code: framework.UnschedulableStatus}
}

func (g *GNumber) Score(ctx context.Context, chunk framework.ChunkInfo, nodeTracker api.NodeTracker, cache *cache.Cache) float32 {
	number := runtime.NumGoroutine()
	return (1 - float32(number)/float32(defaultGoroutineLimit)) * 100
}
