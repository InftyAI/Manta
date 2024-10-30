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
)

const (
	buffSize = 4 * 1024 * 1024 // 4MB buffer
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
	defer file.Close()

	buffer := make([]byte, buffSize)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			_, writeErr := w.Write(buffer[:n])
			if writeErr != nil {
				fmt.Println("Error writing to response:", writeErr)
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
	}
}

func recvChunk(blobPath, snapshotPath, peerName string) error {
	url := fmt.Sprintf("http://%s:8080/sync?path=%s", peerName, blobPath)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Use the same path for different peers.
	file, err := os.Create(blobPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	if err := createSymlink(blobPath, snapshotPath); err != nil {
		return err
	}

	fmt.Println("Chunk synced successfully")
	return nil
}
