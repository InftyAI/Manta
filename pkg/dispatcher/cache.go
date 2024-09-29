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

package dispatcher

import (
	"strconv"
	"sync"

	"github.com/inftyai/manta/pkg/util"
)

const (
	EmptyPlaceholder = ""
)

// cache will maintain the data info of all the peers across cluster.
type cache struct {
	// The embedding locks order must be repoInfo lock -> cache lock, or will lead to deadlock.
	sync.RWMutex
	// Repos with key refers to he repo name, e.g. Llama-3.1-8B-Instruct and
	// value refers to the repo info.
	// If the key is "", which means it's a single chunk.
	repos map[string]*repoInfo
}

type repoInfo struct {
	sync.RWMutex
	// objects with the key refer to the object path and the value refers to
	// the objectInfo.
	// The object could be file only for now, if it's a directory, we'll not
	// record here.
	objects map[string]*objectInfo
}

type objectInfo struct {
	// The path of the file, can be used as an identifier.
	path string
	// chunkNumber refers to the technically the total chunk number
	// of the object, once the number is equal to the length of the
	// chunks, which means the object is downloaded successfully.
	chunkNumber int32
	// The whole chunks to compose a file.
	chunks map[string]*chunkInfo
}

type chunkInfo struct {
	// The chunk name, it's formatted as <hash>--00FF, which means
	// a file can be at most split into 15 * 16 + 15 * 1 = 255 chunks,
	// the OO means the first chunk, the FF means the maximum number.
	name string
	size int32
	// nodes represents the node containers the chunk.
	// Theoretically, the nodes number should be greater than Torrent's replicas
	// and the number wouldn't be too large.
	nodes []string
}

func NewCache() *cache {
	c := cache{
		repos: map[string]*repoInfo{},
	}
	return &c
}

func (c *cache) GetChunkInfo(reponame, objpath, chunkname string) (*chunkInfo, bool) {
	c.RLock()
	repo, exists := c.repos[reponame]
	if !exists {
		c.Unlock()
		return nil, false
	}
	c.Unlock()

	repo.RLock()
	defer repo.Unlock()

	// If repo.objects is nil, the repo should already be deleted.
	obj, exists := repo.objects[objpath]
	if !exists {
		return nil, false
	}

	// if file.chunks is nil, the file should already be deleted.
	chunk, exists := obj.chunks[chunkname]
	return chunk, exists
}

func (c *cache) AddChunk(reponame string, objpath string, cinfo chunkInfo) error {
	c.Lock()
	repo, exists := c.repos[reponame]
	if !exists {
		repo = &repoInfo{objects: map[string]*objectInfo{}}
		c.repos[reponame] = repo
	}
	c.Unlock()

	number, err := strconv.ParseInt(cinfo.name[len(cinfo.name)-2:], 16, 32)
	if err != nil {
		return err
	}

	repo.Lock()
	defer repo.Unlock()

	obj, exists := repo.objects[objpath]
	if !exists {
		obj = &objectInfo{
			path:        objpath,
			chunkNumber: int32(number),
			chunks:      map[string]*chunkInfo{},
		}
		repo.objects[objpath] = obj
	}

	chunk, exists := obj.chunks[cinfo.name]
	if !exists {
		chunk = &chunkInfo{
			name: cinfo.name,
			size: cinfo.size,
		}
		obj.chunks[cinfo.name] = chunk
		return nil
	}

	// update the chunk.
	chunk.size = cinfo.size
	// For each replication, we should only have one node.
	// Or this should be changed.
	nodes := util.SetAdd(chunk.nodes, cinfo.nodes[0])
	chunk.nodes = nodes

	return nil
}

// TODO:
func (c *cache) UpdateChunk(reponame string, cinfo chunkInfo) {
}

func (c *cache) RemoveChunk(reponame, objpath, chunkname, nodename string) {
	c.RLock()
	repo, exists := c.repos[reponame]
	if !exists {
		c.Unlock()
		return
	}
	c.Unlock()

	repo.Lock()
	defer repo.Unlock()

	obj, exists := repo.objects[objpath]
	if !exists {
		return
	}

	chunk, exists := obj.chunks[chunkname]
	if !exists {
		return
	}

	chunk.nodes = util.SetRemove(chunk.nodes, nodename)

	// cleanup
	if len(chunk.nodes) == 0 {
		delete(obj.chunks, chunkname)
		if len(obj.chunks) == 0 {
			delete(repo.objects, objpath)
		}
		if len(repo.objects) == 0 {
			c.Lock()
			delete(c.repos, reponame)
			c.Unlock()
		}
	}
}
