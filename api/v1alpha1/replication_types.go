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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	URI_LOCALHOST = "localhost"
	URI_REMOTE    = "remote"
)

// Target represents the to be replicated file info.
// Source couldn't be nil, but if destination is nil,
// it means to delete the file.
type Target struct {
	// URI represents the file address with different storages, e.g.:
	// 	 - oss://<bucket>.<endpoint>/<path-to-your-file>
	// 	 - localhost://<path-to-your-file>
	// 	 - remote://<node-name>@<path-to-your-file>
	// Localhost means the local host path, remote means the host path of the provided node.
	// Note: if it's a folder, all the files under the folder will be considered,
	// otherwise, only one file will be replicated.
	URI *string `json:"uri,omitempty"`
	// Hub represents the model registry for model downloads.
	// Hub and address are exclusive.
	// +optional
	Hub *Hub `json:"hub,omitempty"`
}

// ReplicationSpec defines the desired state of Replication
type ReplicationSpec struct {
	// NodeName represents which node should do replication.
	NodeName string `json:"nodeName"`
	// ChunkName represents the replicating chunk name.
	ChunkName string `json:"chunkName"`
	// Source represents the source file.
	// Source couldn't be nil.
	Source Target `json:"source"`
	// Destination represents the destination of the file.
	// If destination is nil, it means to delete the file.
	// +optional
	Destination *Target `json:"destination,omitempty"`
	// SizeBytes represents the chunk size.
	SizeBytes int64 `json:"sizeBytes"`
}

type ReplicateState string

const (
	ReplicatingReplicateState ReplicateState = "Replicating"
	ReadyReplicateState       ReplicateState = "Ready"
)

// ReplicationStatus defines the observed state of Replication
type ReplicationStatus struct {
	// Conditions represents the Torrent condition.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Phase represents the current state.
	// +optional
	Phase *string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="node",type=string,JSONPath=".spec.nodeName"
//+kubebuilder:printcolumn:name="phase",type=string,JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Replication is the Schema for the replications API
type Replication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicationSpec   `json:"spec,omitempty"`
	Status ReplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReplicationList contains a list of Replication
type ReplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Replication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Replication{}, &ReplicationList{})
}
