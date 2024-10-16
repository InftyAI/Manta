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

import "os"

// The files length MUST BE THE SAME with blobs, and they should be one-to-one mapping.
// The elements in blobs should not be empty, but if one element in files is empty,
// which means the file is still under downloading.
//
// Note: don't forget to cleanup the folders.
func MockRepo(rootPath string, repoName string, revision string, files []string, blobs []string) error {
	repoPath := rootPath + repoName + "/"
	blobPath := repoPath + "blobs/"
	filePath := repoPath + "snapshots/" + revision + "/"

	if err := os.MkdirAll(blobPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filePath, 0755); err != nil {
		return err
	}

	symlinkPrefix := "../../blobs/"

	for i := 0; i < len(files); i++ {
		if _, err := os.Create(blobPath + blobs[i]); err != nil {
			return err
		}

		if files[i] != "" {
			if err := os.Symlink(symlinkPrefix+blobs[i], filePath+files[i]); err != nil {
				return err
			}
		}
	}

	return nil
}
