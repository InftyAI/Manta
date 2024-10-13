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
	"fmt"
	"strings"
	"sync"

	api "github.com/inftyai/manta/api/v1alpha1"
)

func HandleAddEvent(replication *api.Replication) {
	var wg sync.WaitGroup

	for i := range replication.Spec.Tuples {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handleTuple(&replication.Spec.Tuples[i]); err != nil {
				fmt.Printf("Error handling tuple: %v.\n", err)
			}
		}()
	}

	wg.Wait()
}

func handleTuple(tuple *api.Tuple) error {
	// If destination is nil, the address must not be localhost.
	if tuple.Destination == nil {
		// TODO: Delete OP
		return nil
	}

	// If modelHub != nil, it must be download to the localhost.
	if tuple.Source.ModelHub != nil {
		_, localPath := parseURI(*tuple.Destination.URI)
		if *tuple.Source.ModelHub.Name == api.HUGGINGFACE_MODEL_HUB {
			if err := downloadFromHF(tuple.Source.ModelHub.ModelID, *tuple.Source.ModelHub.Revision, *tuple.Source.ModelHub.Filename, localPath); err != nil {
				return err
			}
			// TODO: handle modelScope
		}
		// TODO: Handle address
	}

	return nil
}

func parseURI(uri string) (host string, address string) {
	splits := strings.Split(uri, "://")
	return splits[0], splits[1]
}
