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

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/pkg/dispatcher/cache"
	"github.com/inftyai/manta/pkg/dispatcher/framework"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ framework.FilterPlugin = &DiskAware{}
var _ framework.ScorePlugin = &DiskAware{}

const (
	// The default memory size is 100Gi.
	defaultSizeLimit = "100Gi"
)

type DiskAware struct{}

func New() (framework.Plugin, error) {
	return &DiskAware{}, nil
}

func (ds *DiskAware) Name() string {
	return "DiskAware"
}

func (ds *DiskAware) Filter(ctx context.Context, chunk framework.ChunkInfo, _ *framework.NodeInfo, nodeTracker api.NodeTracker, cache *cache.Cache) framework.Status {
	nodeName := nodeTracker.Name
	totalSize := cache.NodeTotalSizeBytes(nodeName)

	sizeLimit := sizeLimit(nodeTracker)
	if totalSize+chunk.Size > sizeLimit {
		return framework.Status{Code: framework.UnschedulableStatus}
	}

	cache.Store(nodeName, totalSize)
	return framework.Status{Code: framework.SuccessStatus}
}

func (ds *DiskAware) Score(ctx context.Context, chunkInfo framework.ChunkInfo, _ *framework.NodeInfo, nodeTracker api.NodeTracker, cache *cache.Cache) float32 {
	var totalSize int64

	loadValue := cache.Load(nodeTracker.Name)
	if loadValue == nil {
		totalSize = cache.NodeTotalSizeBytes(nodeTracker.Name)
	} else {
		totalSize = loadValue.(int64)
	}

	sizeLimit := sizeLimit(nodeTracker)
	return (1 - float32(totalSize+chunkInfo.Size)/float32(sizeLimit)) * 100
}

func sizeLimit(nt api.NodeTracker) int64 {
	limit := defaultSizeLimit
	if nt.Spec.SizeLimit != nil {
		limit = *nt.Spec.SizeLimit
	}

	// TODO: we'll validate the value in the webhooks to make sure no panic here.
	value := resource.MustParse(limit)
	return value.Value()
}
