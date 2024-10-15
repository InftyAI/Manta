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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api "github.com/inftyai/manta/api/v1alpha1"
)

const (
	syncDuration = 1 * time.Minute
)

var (
	logger logr.Logger
)

func BackgroundTasks(ctx context.Context, c client.Client) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger = ctrl.Log.WithName("Background")

	// Sync the disk chunk infos to the nodeTracker.
	go syncChunks(ctx, c)
}

func syncChunks(ctx context.Context, c client.Client) {
	forFunc := func() error {
		attempts := 0
		for {
			attempts += 1
			if err := findOrCreateNodeTracker(ctx, c); err != nil {
				// fmt.Printf("failed to create nodeTracker: %v, retry.", err)
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
		if err := forFunc(); err != nil {
			// If happens, which means the cluster is unstable.
			logger.Error(err, "Failed to create nodeTracker")
		} else {
			logger.Info("Syncing the chunks")
			// TODO
		}

		time.Sleep(syncDuration)
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
		return c.Create(newCtx, &nodeTracker)
	}

	return nil
}
