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

package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ObjectBody struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Oid  string `json:"oid"`
	Size int64  `json:"size"`
}

// TODO: support modelScope as well.
func ListRepoObjects(repoID string, revision string) (bodies []*ObjectBody, err error) {
	url := fmt.Sprintf("https://huggingface.co/api/models/%s/tree/%s", repoID, revision)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repo files: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	info := []*ObjectBody{}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return info, nil
}
