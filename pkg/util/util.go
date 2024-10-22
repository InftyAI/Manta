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
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
)

func GenerateName(prefix string) (string, error) {
	if prefix == "" {
		return "", errors.New("no prefix")
	}

	const suffixLength = 5
	randomBytes := make([]byte, suffixLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	suffix := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	suffix = strings.ToLower(suffix)

	return prefix + "--" + suffix[0:suffixLength], nil
}

// toDelete includes string in old but not in new,
// toAdd includes string in new but not in old.
func SliceDiff(old []string, new []string) (toDelete []string, toAdd []string) {
	for _, s := range old {
		if !SliceIn(new, s) {
			toDelete = append(toDelete, s)
		}
	}

	for _, s := range new {
		if !SliceIn(old, s) {
			toAdd = append(toAdd, s)
		}
	}
	return
}

func SliceIn(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
