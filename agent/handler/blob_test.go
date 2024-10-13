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

func TestBlobExists(t *testing.T) {
	if err := os.Setenv("WORKSPACE", "../../tmp/workspace/models/"); err != nil {
		t.Fail()
	}

	path := "../../tmp/workspace/models/fakeRepo/blobs/"
	filename := "fakeChunk"

	if blobExists("fakeRepo", "fakeChunk") {
		t.Error("blob should be not exist")
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("run mkdir failed: %v", err)
	}

	file, err := os.Create(path + filename)
	if err != nil {
		t.Error("failed to create file")
	}
	defer func() {
		_ = file.Close()
	}()
	defer func() {
		_ = os.RemoveAll("../../tmp/workspace")
	}()

	if !blobExists("fakeRepo", "fakeChunk") {
		t.Error("blob should be exist")
	}
}
