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

import "testing"

func TestGenerateName(t *testing.T) {
	testCases := []struct {
		name       string
		prefix     string
		wantLength int32
		wantErr    bool
	}{
		{
			name:       "with non-empty prefix",
			prefix:     "prefix",
			wantLength: 13,
			wantErr:    false,
		},
		{
			name:       "empty prefix",
			prefix:     "",
			wantLength: 0,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, err := GenerateName(tc.prefix)
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(name) != int(tc.wantLength) {
				t.Fatalf("unexpected length, name: %s", name)
			}
		})
	}
}
