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

func TestSetContains(t *testing.T) {
	testCases := []struct {
		name     string
		values   []string
		value    string
		contains bool
	}{
		{
			name:     "contain the value",
			values:   []string{"foo", "bar"},
			value:    "foo",
			contains: true,
		},
		{
			name:     "do not contain",
			values:   []string{"foo", "bar"},
			value:    "fuz",
			contains: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotContains := SetContains(tc.values, tc.value)
			if gotContains != tc.contains {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestSetAdd(t *testing.T) {
	testCases := []struct {
		name       string
		values     []string
		value      string
		wantValues []string
	}{
		{
			name:       "contain the value",
			values:     []string{"foo", "bar"},
			value:      "foo",
			wantValues: []string{"bar", "foo"},
		},
		{
			name:       "do not contain",
			values:     []string{"foo", "bar"},
			value:      "fuz",
			wantValues: []string{"foo", "bar", "fuz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := SetAdd(tc.values, tc.value)
			if diff := cmp.Diff(values, tc.wantValues); diff != "" {
				t.Fatalf("unexpected result, want %v, got %v", tc.wantValues, values)
			}
		})
	}
}

func TestSetRemove(t *testing.T) {
	testCases := []struct {
		name       string
		values     []string
		value      string
		wantValues []string
	}{
		{
			name:       "contain the value",
			values:     []string{"foo", "bar"},
			value:      "foo",
			wantValues: []string{"bar"},
		},
		{
			name:       "do not contain",
			values:     []string{"foo", "bar"},
			value:      "fuz",
			wantValues: []string{"foo", "bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := SetRemove(tc.values, tc.value)
			if diff := cmp.Diff(values, tc.wantValues); diff != "" {
				t.Fatalf("unexpected result, want %v, got %v", tc.wantValues, values)
			}
		})
	}
}
