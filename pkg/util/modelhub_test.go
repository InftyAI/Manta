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
	"testing"
)

func TestListRepoFiles(t *testing.T) {
	testCases := []struct {
		name     string
		repoID   string
		revision string
		wantErr  bool
	}{
		{
			name:     "right repo",
			repoID:   "Qwen/Qwen2-7B-Instruct",
			revision: "main",
		},
		{
			name:     "non-existence repo",
			repoID:   "QQwen/Qwen2-7B-Instruct",
			wantErr:  true,
			revision: " main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects, err := ListRepoObjects(tc.repoID, tc.revision)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected err: %v", err)
			}

			if err == nil && tc.wantErr {
				t.Fatal("no error returned")
			}

			if err == nil && len(objects) == 0 {
				t.Fatal("empty objects")
			}
		})
	}
}
