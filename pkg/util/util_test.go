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

	"github.com/google/go-cmp/cmp"
)

func TestGenerateName(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "",
		},
		{
			name: "node",
			want: "f8e96",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateName(tc.name)
			if tc.want != got {
				t.Fatalf("unexpected value, want: %s, got: %s", tc.want, got)
			}
		})
	}
}

func TestSliceDiff(t *testing.T) {
	testCases := []struct {
		name          string
		oldSlice      []string
		newSlice      []string
		toDeleteSlice []string
		toAddSlice    []string
	}{
		{
			name:          "empty oldSlice",
			oldSlice:      nil,
			newSlice:      []string{"string1", "string2"},
			toDeleteSlice: nil,
			toAddSlice:    []string{"string1", "string2"},
		},
		{
			name:          "empty newSlice",
			oldSlice:      []string{"string1", "string2"},
			newSlice:      nil,
			toDeleteSlice: []string{"string1", "string2"},
			toAddSlice:    nil,
		},
		{
			name:          "mixed slice",
			oldSlice:      []string{"string1", "string2"},
			newSlice:      []string{"string2", "string3"},
			toDeleteSlice: []string{"string1"},
			toAddSlice:    []string{"string3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotDeleteSlice, gotAddSlice := SliceDiff(tc.oldSlice, tc.newSlice)
			if diff := cmp.Diff(tc.toDeleteSlice, gotDeleteSlice); diff != "" {
				t.Errorf("unexpected to delete slice, diff: %v", diff)
			}
			if diff := cmp.Diff(tc.toAddSlice, gotAddSlice); diff != "" {
				t.Errorf("unexpected to add slice, diff: %v", diff)
			}
		})
	}
}
