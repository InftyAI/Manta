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

package task

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/inftyai/manta/agent/pkg/util"
)

func Test_walkThroughChunks(t *testing.T) {
	rootPath := "../../../tmp/models/"
	defer func() {
		_ = os.RemoveAll(rootPath)
	}()

	if err := util.MockRepo(rootPath, "model-1", "main", []string{"file1", "file2", "file3"}, []string{"blob1", "blob2", "blob-same"}); err != nil {
		t.Error(err)
	}

	if err := util.MockRepo(rootPath, "model-2", "master", []string{"fileA", "fileB", "fileC"}, []string{"blobA", "blobB", "blob-same"}); err != nil {
		t.Error(err)
	}

	chunks, err := walkThroughChunks(rootPath)
	if err != nil {
		t.Error(err)
	}

	wantFiles := []chunkInfo{{Name: "blob1"}, {Name: "blob2"}, {Name: "blob-same"}, {Name: "blobA"}, {Name: "blobB"}}

	if diff := cmp.Diff(chunks, wantFiles); diff != "" {
		t.Errorf("unexpected files, diff %v", diff)
	}
}
