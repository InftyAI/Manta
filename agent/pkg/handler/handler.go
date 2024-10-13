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

package handler

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/inftyai/manta/api/v1alpha1"
)

func HandleReplication(logger logr.Logger, client client.Client, replication *api.Replication) (succeeded bool, stateChanged bool) {
	var wg sync.WaitGroup
	var errCount int32

	logger.Info("start to handle Replication", "Replication", klog.KObj(replication))

	for i := range replication.Spec.Tuples {
		if *replication.Spec.Tuples[i].State == api.FinishedStateType {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handleTuple(logger, &replication.Spec.Tuples[i]); err != nil {
				logger.Error(err, "failed to handle Tuple")
				atomic.AddInt32(&errCount, 1)
			} else {
				condition := api.FinishedStateType
				replication.Spec.Tuples[i].State = (*api.StateType)(&condition)
				stateChanged = true
			}
		}()
	}

	wg.Wait()
	return errCount == 0, stateChanged
}

func handleTuple(logger logr.Logger, tuple *api.Tuple) error {
	// If destination is nil, the address must not be localhost.
	if tuple.Destination == nil {
		// TODO: Delete OP
		return nil
	}

	// If modelHub != nil, it must be download to the localhost.
	if tuple.Source.ModelHub != nil {
		_, localPath := parseURI(*tuple.Destination.URI)
		if *tuple.Source.ModelHub.Name == api.HUGGINGFACE_MODEL_HUB {
			if err := downloadFromHF(tuple.Source.ModelHub.ModelID, *tuple.Source.ModelHub.Revision, *tuple.Source.ModelHub.Filename, localPath); err != nil {
				return err
			}
			// TODO: handle modelScope
		}
		// TODO: Handle address

		logger.Info("download file successfully", "file", *tuple.Source.ModelHub.Filename)
	}

	return nil
}

func parseURI(uri string) (host string, address string) {
	splits := strings.Split(uri, "://")
	return splits[0], splits[1]
}
