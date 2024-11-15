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

	"github.com/inftyai/manta/api"
)

const (
	buffSize = 10 * 1024 * 1024 // 10MB buffer
)

// SendChunk will send the chunk content via http request.
func SendChunk(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	buffer := make([]byte, buffSize)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println("Error reading file")
				http.Error(w, "Error reading file", http.StatusInternalServerError)
				return
			}
		}

		if n > 0 {
			_, writeErr := w.Write(buffer[:n])
			if writeErr != nil {
				fmt.Println("Error writing to response:", writeErr)
				http.Error(w, "Error writing to response", http.StatusInternalServerError)
				return
			}
		}
	}
}

func recvChunk(blobPath, snapshotPath, addr string) error {
	url := fmt.Sprintf("http://%s:%s/sync?path=%s", addr, api.HttpPort, blobPath)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err := os.MkdirAll(filepath.Dir(blobPath), os.ModePerm); err != nil {
		return err
	}

	// Use the same path for different peers.
	file, err := os.Create(blobPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	if err := createSymlink(blobPath, snapshotPath); err != nil {
		return err
	}

	return nil
}
