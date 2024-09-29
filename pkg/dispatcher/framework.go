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

package dispatcher

// Download represents the methods to download a chunk.
type Download interface {
	Framework
}

// Sync represents the methods to sync a chunk.
type Sync interface {
	Framework
}

// Framework represents the algo about how to pick the candidates among all the peers.
type Framework interface {
	// RegisterPlugins will register the plugins to run.
	RegisterPlugins([]string)
	// RunFilterPlugins will filter out unsatisfied peers.
	RunFilterPlugins()
	// RunScorePlugins will calculate the scores of all the peers.
	RunScorePlugins()
}
