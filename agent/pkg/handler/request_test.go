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
	"os"
	"testing"
)

func Test_downloadFromHF(t *testing.T) {
	testCases := []struct {
		name         string
		modelID      string
		revision     string
		path         string
		downloadPath string
		wantError    bool
	}{
		{
			name:         "normal download",
			modelID:      "Qwen/Qwen2.5-72B-Instruct",
			revision:     "main",
			path:         "LICENSE",
			downloadPath: "../../../tmp/LICENSE",
			wantError:    false,
		},
		{
			name:         "unknown revision",
			modelID:      "Qwen/Qwen2.5-72B-Instruct",
			revision:     "master", // unknown branch
			path:         "LICENSE",
			downloadPath: "../../../tmp/LICENSE",
			wantError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotError := downloadFromHF(tc.modelID, tc.revision, tc.path, tc.downloadPath)
			defer func() {
				_ = os.RemoveAll(tc.downloadPath)
			}()

			if tc.wantError && gotError == nil {
				t.Error("expected error here")
			}
			if !tc.wantError {
				if _, err := os.Stat(tc.downloadPath); err != nil {
					t.Error("expected file downloaded successfully")
				}
			}
		})
	}
}
