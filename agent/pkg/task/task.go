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

package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cons "github.com/inftyai/manta/api"
	api "github.com/inftyai/manta/api/v1alpha1"
)

const (
	syncDuration = 5 * time.Minute

	workspace = cons.DefaultWorkspace
)

func BackgroundTasks(ctx context.Context, c client.Client) {
	// Sync the disk chunk infos to the nodeTracker.
	go syncChunks(ctx, c)
}

func syncChunks(ctx context.Context, c client.Client) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("Background tasks")

	forFunc := func(ctx context.Context) error {
		attempts := 0
		for {
			attempts += 1
			if err := findOrCreateNodeTracker(ctx, c); err != nil {
				logger.Error(err, "Failed to create nodeTracker, retry...")

				if attempts > 10 {
					return fmt.Errorf("reach the maximum attempt times")
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}

			break
		}
		return nil
	}

	for {
		// To avoid context memory escape.
		ctx, cancel := context.WithCancel(ctx)

		if err := forFunc(ctx); err != nil {
			// If happens, which means the cluster is unstable.
			logger.Error(err, "Failed to create nodeTracker")
		} else {
			logger.Info("Syncing the chunks")

			if chunks, err := walkThroughChunks(workspace); err != nil {
				logger.Error(err, "Failed to walk through chunks")
			} else {
				nodeTracker := &api.NodeTracker{}
				if err := c.Get(ctx, types.NamespacedName{Name: os.Getenv("NODE_NAME")}, nodeTracker); err != nil {
					logger.Error(err, "Failed to get nodeTracker", "nodeTracker", os.Getenv("NODE_NAME"))
				} else {
					UpdateChunks(nodeTracker, chunks)
					if err := c.Update(ctx, nodeTracker); err != nil {
						logger.Error(err, "Failed to update nodeTracker", "NodeTracker", nodeTracker.Name)
					}
				}
			}
		}

		cancel()
		time.Sleep(syncDuration)
	}
}

func UpdateChunks(nt *api.NodeTracker, chunks []chunkInfo) {
	if len(chunks) == 0 {
		nt.Spec.Chunks = nil
		return
	}

	nt.Spec.Chunks = make([]api.ChunkTracker, 0, len(chunks))
	for _, chunk := range chunks {
		nt.Spec.Chunks = append(nt.Spec.Chunks,
			api.ChunkTracker{
				ChunkName: chunk.Name,
				SizeBytes: chunk.SizeBytes,
			},
		)
	}
}

func findOrCreateNodeTracker(ctx context.Context, c client.Client) error {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return fmt.Errorf("NODE_NAME not exists")
	}

	nodeTracker := api.NodeTracker{}

	if err := c.Get(newCtx, types.NamespacedName{Name: nodeName}, &nodeTracker); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		var node corev1.Node
		if err := c.Get(newCtx, types.NamespacedName{Name: nodeName}, &node); err != nil {
			return err
		}

		nodeTracker.Name = nodeName
		nodeTracker.Labels = node.Labels
		nodeTracker.OwnerReferences = []v1.OwnerReference{
			{
				Kind:               "Node",
				APIVersion:         "v1",
				Name:               node.Name,
				UID:                node.UID,
				BlockOwnerDeletion: ptr.To(true),
				Controller:         ptr.To(true),
			},
		}

		sizeLimit := os.Getenv("SIZE_LIMIT")
		if sizeLimit != "" {
			nodeTracker.Spec.SizeLimit = ptr.To[string](sizeLimit)
		}

		return c.Create(newCtx, &nodeTracker)
	}

	return nil
}

type chunkInfo struct {
	Name      string
	SizeBytes int64
}

func walkThroughChunks(path string) (chunks []chunkInfo, err error) {
	fileMap := make(map[string]struct{})

	repos, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}

		snapshotPath := path + repo.Name() + "/snapshots/"
		if _, err := os.Stat(snapshotPath); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}

		revisions, err := os.ReadDir(snapshotPath)
		if err != nil {
			return nil, err
		}

		for _, revision := range revisions {
			revisionPath := snapshotPath + revision.Name() + "/"

			if !revision.IsDir() {
				continue
			}

			files, err := os.ReadDir(revisionPath)
			if err != nil {
				return nil, err
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				filePath := revisionPath + file.Name()
				targetPath, err := os.Readlink(filePath)
				if err != nil {
					return nil, err
				}
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					return nil, err
				}

				chunkName := filepath.Base(targetPath)

				// To avoid duplicated files
				if _, ok := fileMap[chunkName]; !ok {
					chunks = append(chunks, chunkInfo{
						Name:      chunkName,
						SizeBytes: fileInfo.Size(),
					})
					fileMap[chunkName] = struct{}{}
				}
			}
		}

	}

	return chunks, nil
}
