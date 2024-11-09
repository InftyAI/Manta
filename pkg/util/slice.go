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
	"sort"
)

func SetContains[T comparable](values []T, value T) bool {
	for i := range values {
		if value == values[i] {
			return true
		}
	}

	return false
}

func SetRemove(values []string, value string) []string {
	newValues := []string{}
	for i := range values {
		if value == values[i] {
			continue
		}
		newValues = append(newValues, values[i])
	}
	return newValues
}

func SetAdd(values []string, value string) []string {
	newValues := []string{}
	for i := range values {
		if value == values[i] {
			continue
		}
		newValues = append(newValues, values[i])
	}
	return append(newValues, value)
}

// Get the top-n indices from the arr.
func TopNIndices(arr []float32, n int) []int32 {
	if len(arr) <= n {
		indices := []int32{}
		for i := 0; i < len(arr); i++ {
			indices = append(indices, int32(i))
		}
		return indices
	}

	type indexedValue struct {
		value float32
		index int
	}

	indexedArr := make([]indexedValue, len(arr))
	for i, v := range arr {
		indexedArr[i] = indexedValue{value: v, index: i}
	}

	// Sort by descend.
	sort.Slice(indexedArr, func(i, j int) bool {
		return indexedArr[i].value > indexedArr[j].value
	})

	topIndices := make([]int32, n)
	for i := 0; i < n; i++ {
		topIndices[i] = int32(indexedArr[i].index)
	}

	return topIndices
}
