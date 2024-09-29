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

func SetContains(values []string, value string) bool {
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
