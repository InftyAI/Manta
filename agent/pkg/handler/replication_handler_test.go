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
	"context"
	"os"
	"testing"

	"github.com/inftyai/manta/test/util/wrapper"
)

func TestHandleReplication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		_ = os.RemoveAll("../../../tmp/replication")
	}()

	// verify download chunk
	toCreateReplication := wrapper.MakeReplication("replication").
		SourceOfHub("Huggingface", "Qwen/Qwen2.5-72B-Instruct", "main", "LICENSE").
		DestinationOfURI("localhost://../../../tmp/replication/models/Qwen--Qwen2.5-72B-Instruct/blobs/LICENSE-chunk").
		Obj()
	if err := HandleReplication(ctx, nil, toCreateReplication); err != nil {
		t.Errorf("failed to handle Replication: %v", err)
	}

	targetPath := "../../../tmp/replication/models/Qwen--Qwen2.5-72B-Instruct/snapshots/main/LICENSE"
	fileInfo, err := os.Lstat(targetPath)
	if err != nil {
		t.Errorf("failed to list file")
	}

	if fileInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
		t.Errorf("file should be a symlink")
	}

	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("file doesn't exists")
	}

	// verify delete chunk
	toDeleteReplication := wrapper.MakeReplication("replication").
		SourceOfURI("localhost://../../../tmp/replication/models/Qwen--Qwen2.5-72B-Instruct/snapshots/main/LICENSE").
		Obj()
	if err := HandleReplication(ctx, nil, toDeleteReplication); err != nil {
		t.Errorf("failed to handle Replication: %v", err)
	}

	_, err = os.Lstat(targetPath)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("file should not exists")
	}

	_, err = os.Lstat("../../../tmp/replication/models/Qwen--Qwen2.5-72B-Instruct/blobs/LICENSE-chunk")
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("file should not exists")
	}
}
