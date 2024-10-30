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
	"sync"

	api "github.com/inftyai/manta/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Cache will maintain the data info of all the peers across cluster.
type Cache struct {
	sync.RWMutex
	// chunks with key refers to the chunk name and value refers to the chunk info.
	chunks map[string]*ChunkInfo
	// nodes with the key refers to the node name and the value refers to the chunk names it hosts.
	nodes map[string]sets.Set[string]

	// These fileds are only used in dispatching, will be set in snapshot.
	// No concurrency happens in dispatching right now, so no need to consider the lock right now,
	// but once concurrency introduced, we may change to use sync.Map or something similar.
	state map[string]interface{}
}

type ChunkInfo struct {
	Name string
	// nodes represents the node contains the chunk.
	// Nodes is empty when part of the cache.nodes.
	Nodes sets.Set[string]
	// SizeBytes represents the chunk size.
	SizeBytes int64
}

func NewCache() *Cache {
	c := Cache{
		chunks: make(map[string]*ChunkInfo),
		nodes:  make(map[string]sets.Set[string]),
	}
	return &c
}

func (c *Cache) AddChunks(chunks []api.ChunkTracker, nodename string) {
	c.Lock()
	defer c.Unlock()

	chunkNames, ok := c.nodes[nodename]
	if !ok {
		chunkNames = sets.New[string]()
		c.nodes[nodename] = chunkNames
	}

	for _, chunk := range chunks {
		chunkNames.Insert(chunk.ChunkName)

		if info, ok := c.chunks[chunk.ChunkName]; ok {
			info.Nodes.Insert(nodename)
			continue
		}

		c.chunks[chunk.ChunkName] = &ChunkInfo{
			Name:      chunk.ChunkName,
			Nodes:     sets.New(nodename),
			SizeBytes: chunk.SizeBytes,
		}
	}
}

// DeleteChunks will delete chunks from one node.
func (c *Cache) DeleteChunks(chunks []api.ChunkTracker, nodename string) {
	c.Lock()
	defer c.Unlock()

	node := c.nodes[nodename]

	for _, chunk := range chunks {
		if info, ok := c.chunks[chunk.ChunkName]; ok {
			info.Nodes.Delete(nodename)
			if len(info.Nodes) == 0 {
				delete(c.chunks, chunk.ChunkName)
			}
		}

		// node should not be nil, just in case.
		if node != nil {
			node.Delete(chunk.ChunkName)
		}
	}
}

// Maybe we can add a field to nodes to represents the totalSize and calculated when
// adding or removing.
func (c *Cache) NodeTotalSizeBytes(nodename string) (size int64) {
	c.RLock()
	defer c.RUnlock()

	chunks, ok := c.nodes[nodename]
	if !ok {
		return 0
	}

	for chunk := range chunks {
		if c, ok := c.chunks[chunk]; ok {
			size += c.SizeBytes
		}
	}
	return
}

func (c *Cache) ChunkNodes(chunkname string) []string {
	c.RLock()
	defer c.RUnlock()

	info, ok := c.chunks[chunkname]
	if !ok {
		return nil
	}

	return info.Nodes.UnsortedList()
}

func (c *Cache) ChunkExist(chunkname string) bool {
	c.RLock()
	defer c.RUnlock()

	_, ok := c.chunks[chunkname]
	return ok
}

func (c *Cache) ChunkExistInNode(nodename, chunkname string) bool {
	c.RLock()
	defer c.RUnlock()

	chunks, ok := c.nodes[nodename]
	if !ok {
		return false
	}
	return chunks.Has(chunkname)
}

// Snapshot is called before dispatching.
func (c *Cache) Snapshot() *Cache {
	c.Lock()
	defer c.Unlock()

	newCache := &Cache{
		chunks: make(map[string]*ChunkInfo, len(c.chunks)),
		nodes:  make(map[string]sets.Set[string], len(c.nodes)),
	}

	for k, v := range c.chunks {
		newNodes := v.Nodes.Clone()
		newChunkInfo := ChunkInfo{
			Name:      v.Name,
			Nodes:     newNodes,
			SizeBytes: v.SizeBytes,
		}
		newCache.chunks[k] = &newChunkInfo
	}

	for k, v := range c.nodes {
		chunkNames := v.Clone()
		newCache.nodes[k] = chunkNames
	}

	newCache.state = make(map[string]interface{})
	return newCache
}

func (c *Cache) Store(k string, v interface{}) {
	c.Lock()
	defer c.Unlock()
	c.state[k] = v
}

func (c *Cache) Load(k string) interface{} {
	c.Lock()
	defer c.Unlock()
	return c.state[k]
}
