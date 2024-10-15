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
	"fmt"
	"os"
	"path/filepath"
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

	var localPath, revision, filename, targetPath string

	// If modelHub != nil, it must be download to the localhost.
	if tuple.Source.ModelHub != nil {
		_, localPath = parseURI(*tuple.Destination.URI)
		revision = *tuple.Source.ModelHub.Revision
		filename = *tuple.Source.ModelHub.Filename
		splits := strings.Split(localPath, "/blobs/")
		targetPath = splits[0] + "/snapshots/" + revision + "/" + filename

		// symlink exists means already downloaded.
		if _, err := os.Stat(targetPath); err == nil {
			logger.Info("file already downloaded", "file", filename)
			return nil
		}

		if *tuple.Source.ModelHub.Name == api.HUGGINGFACE_MODEL_HUB {
			if err := downloadFromHF(tuple.Source.ModelHub.ModelID, revision, filename, localPath); err != nil {
				return err
			}
			// TODO: handle modelScope
		}
		// TODO: Handle address
		logger.Info("download file successfully", "file", filename)
	}

	// symlink can helps to validate the file is downloaded successfully.
	// TODO: once we support split a file to several chunks, the targetPath should be
	// changed here, such as targetPath-0001.
	if err := createSymlink(localPath, targetPath); err != nil {
		logger.Error(err, "failed to create symlink")
		return err
	}

	logger.Info("create symlink successfully", "file", filename)
	return nil
}

// localPath looks like: /workspace/models/Qwen--Qwen2-0.5B-Instruct-GGUF/blobs/8b08b8632419bd6d7369362945b5976c7f47b1c1--0001
// symlink file looks like: /mnt/models/Qwen--Qwen2-0.5B-Instruct-GGUF/snapshots/main/qwen2-0_5b-instruct-q5_k_m.gguf
func createSymlink(localPath, targetPath string) error {
	dir := filepath.Dir(targetPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	if _, err := os.Lstat(targetPath); err == nil {
		err = os.Remove(targetPath)
		if err != nil {
			return err
		}
	}

	splits := strings.Split(localPath, "/blobs/")
	if len(splits) != 2 {
		return fmt.Errorf("unexpected localPath: %s", localPath)
	}

	// TODO: once we support file-chunks, we may need to refactor the file name,
	// like merges.txt.chunks--0002, merges.txt.chunks--0102.
	// Use relative link to avoid the host folder is different with the container folder,
	// one is /mnt/models, another is /workspace/models.
	sourcePath := "../.." + "/blobs/" + splits[1]
	return os.Symlink(sourcePath, targetPath)
}

func parseURI(uri string) (host string, address string) {
	splits := strings.Split(uri, "://")
	return splits[0], splits[1]
}
