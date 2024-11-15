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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/inftyai/manta/api/v1alpha1"
)

// This only happens when replication not ready.
func HandleReplication(ctx context.Context, client client.Client, replication *api.Replication) error {
	// If destination is nil, the address must not be localhost.
	if replication.Spec.Destination == nil {
		return deleteChunk(ctx, replication)
	}

	if replication.Spec.Source.Hub != nil {
		return downloadChunk(ctx, replication)
	}

	if replication.Spec.Source.URI != nil {
		host, _ := parseURI(*replication.Spec.Source.URI)
		// TODO: handel other uris.
		if host != api.URI_LOCALHOST {
			return syncChunk(ctx, client, replication)
		}
		// TODO: handle uri with object store.
	}
	return nil
}

func downloadChunk(ctx context.Context, replication *api.Replication) error {
	logger := log.FromContext(ctx)

	var blobPath, revision, filename, targetPath string

	// If hub != nil, it must be download to the localhost.
	if replication.Spec.Source.Hub != nil {
		_, blobPath = parseURI(*replication.Spec.Destination.URI)
		revision = *replication.Spec.Source.Hub.Revision
		filename = *replication.Spec.Source.Hub.Filename
		splits := strings.Split(blobPath, "/blobs/")
		targetPath = splits[0] + "/snapshots/" + revision + "/" + filename

		// symlink exists means already downloaded.
		if _, err := os.Stat(targetPath); err == nil {
			logger.Info("file already downloaded", "file", filename)
			return nil
		}

		if *replication.Spec.Source.Hub.Name == api.HUGGINGFACE_MODEL_HUB {
			logger.Info("Start to download file from Huggingface Hub", "file", filename)
			if err := downloadFromHF(replication.Spec.Source.Hub.RepoID, revision, filename, blobPath); err != nil {
				return err
			}
			// TODO: handle modelScope
		}
		// TODO: Handle address
	}

	// symlink can help to validate the file is downloaded successfully.
	// TODO: once we support split a file to several chunks, the targetPath should be
	// changed here, such as targetPath-0001.
	if err := createSymlink(blobPath, targetPath); err != nil {
		logger.Error(err, "failed to create symlink")
		return err
	}

	logger.Info("download chunk successfully", "file", filename)
	return nil
}

func syncChunk(ctx context.Context, client client.Client, replication *api.Replication) error {
	logger := log.FromContext(ctx)

	logger.Info("start to sync chunks", "Replication", klog.KObj(replication))

	// The source URI looks like remote://node@<path-to-your-file>
	sourceSplits := strings.Split(*replication.Spec.Source.URI, "://")
	addresses := strings.Split(sourceSplits[1], "@")
	nodeName, blobPath := addresses[0], addresses[1]
	addr, err := peerAddr(ctx, client, nodeName)
	if err != nil {
		return err
	}

	// The destination URI looks like localhost://<path-to-your-file>
	destSplits := strings.Split(*replication.Spec.Destination.URI, "://")

	if err := recvChunk(blobPath, destSplits[1], addr); err != nil {
		logger.Error(err, "failed to sync chunk")
		return err
	}

	logger.Info("sync chunk successfully", "Replication", klog.KObj(replication), "target", destSplits[0])
	return nil
}

func deleteChunk(ctx context.Context, replication *api.Replication) error {
	logger := log.FromContext(ctx)
	logger.Info("try to delete chunk", "Replication", replication.Name, "chunk", replication.Spec.ChunkName)
	splits := strings.Split(*replication.Spec.Source.URI, "://")
	if err := deleteSymlinkAndTarget(splits[1]); err != nil {
		logger.Error(err, "failed to delete chunk", "Replication", klog.KObj(replication), "chunk", replication.Spec.ChunkName)
	}
	return nil
}

// local(real) file looks like: /workspace/models/Qwen--Qwen2-0.5B-Instruct-GGUF/blobs/8b08b8632419bd6d7369362945b5976c7f47b1c1--0001
// target file looks like /workspace/models/Qwen--Qwen2-0.5B-Instruct-GGUF/snapshots/main/qwen2-0_5b-instruct-q5_k_m.gguf
// the symlink of target file looks like ../../blobs/8b08b8632419bd6d7369362945b5976c7f47b1c1--0001
func createSymlink(localPath, targetPath string) error {
	// This could happen like force delete a Torrent but downloading is still on the way,
	// then the blob file is deleted, in this situation, we should not create the symlink.
	if _, err := os.Stat(localPath); err != nil {
		return err
	}

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

func deleteSymlinkAndTarget(symlinkPath string) error {
	targetPath, err := filepath.EvalSymlinks(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink: %v", err)
	}

	if err := os.Remove(symlinkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %v", err)
	}

	if _, err := os.Stat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove target file: %v", err)
		}
		fmt.Printf("Target file %s removed.\n", targetPath)
	} else if os.IsNotExist(err) {
		fmt.Printf("Target file %s does not exist.\n", targetPath)
	} else {
		return fmt.Errorf("failed to check target file: %v", err)
	}

	return nil
}

func parseURI(uri string) (host string, address string) {
	splits := strings.Split(uri, "://")
	return splits[0], splits[1]
}

func peerAddr(ctx context.Context, c client.Client, nodeName string) (string, error) {
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
	listOptions := &client.ListOptions{
		FieldSelector: fieldSelector,
		// HACK: the label is hacked, we must set it explicitly in the daemonSet.
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": "manta-agent"}),
	}

	pods := corev1.PodList{}
	if err := c.List(ctx, &pods, listOptions); err != nil {
		return "", err
	}

	if len(pods.Items) != 1 {
		return "", fmt.Errorf("got more than one pod per node for daemonSet")
	}

	return pods.Items[0].Status.PodIP, nil
}
