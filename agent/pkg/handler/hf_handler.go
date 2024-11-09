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

	"github.com/inftyai/manta/agent/pkg/util"
)

const (
	maxAttempts = 10
)

// The downloadPath is the full path, like: /workspace/models/Qwen--Qwen2-7B-Instruct/blobs/20024bfe7c83998e9aeaf98a0cd6a2ce6306c2f0--0001
func downloadFromHF(modelID, revision, path string, downloadPath string) error {
	// Example: "https://huggingface.co/Qwen/Qwen2.5-72B-Instruct/resolve/main/model-00031-of-00037.safetensors"
	url := fmt.Sprintf("%s/%s/resolve/%s/%s", hfEndpoint(), modelID, revision, path)
	token := hfToken()

	attempts := 0
	for {

		attempts += 1

		if err := util.DownloadFileWithResume(url, downloadPath, token); err != nil {
			if attempts > maxAttempts {
				return fmt.Errorf("reach maximum download attempts for %s, err: %v", downloadPath, err)
			}
			continue
		}
		break
	}

	return nil
}

func hfEndpoint() string {
	hfEndpoint := "https://huggingface.co"
	if endpoint := os.Getenv("HF_ENDPOINT"); endpoint != "" {
		hfEndpoint = endpoint
	}
	return hfEndpoint
}

func hfToken() string {
	if token := os.Getenv("HF_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("HUGGING_FACE_HUB_TOKEN"); token != "" {
		return token
	}
	return ""
}
