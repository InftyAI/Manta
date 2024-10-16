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
	"io"
	"net/http"
	"os"
	"path/filepath"
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

		if err := downloadFileWithResume(url, downloadPath, token); err != nil {
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

func downloadFileWithResume(url string, file string, token string) error {
	dir := filepath.Dir(file)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	out, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	fileInfo, err := out.Stat()
	if err != nil {
		return err
	}
	existingFileSize := fileInfo.Size()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if existingFileSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingFileSize))
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !(resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusRequestedRangeNotSatisfiable ||
		resp.StatusCode == http.StatusPartialContent) {
		return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	// File is already downloaded. Should we be more cautious here?
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return nil
	}

	// If the server doesn't support partial download, return error.
	if resp.StatusCode != http.StatusPartialContent && existingFileSize > 0 {
		return fmt.Errorf("server doesn't support resuming downloads, status: %s", resp.Status)
	}

	_, err = out.Seek(existingFileSize, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
