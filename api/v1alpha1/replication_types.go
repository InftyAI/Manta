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

// Target represents the to be replicated file info.
// Source couldn't be nil, but if destination is nil,
// it means to delete the file.
type Target struct {
	// URI represents the file address with different storage.
	// - oss://<bucket>.<endpoint>/<path-to-your-file>
	// - localhost://<path-to-your-file>
	URI *string `json:"uri,omitempty"`
	// ModelHub represents the model registry for model downloads.
	// ModelHub and address are exclusive.
	// +optional
	ModelHub *ModelHub `json:"modelHub,omitempty"`
}

// Tuple represents a pair of source and destination.
type Tuple struct {
	// Source represents the source file.
	// Source couldn't be nil.
	Source Target `json:"source"`
	// Destination represents the destination of the file.
	// If destination is nil, it means to delete the file.
	// +optional
	Destination *Target `json:"destination,omitempty"`
}

// ReplicationSpec defines the desired state of Replication
type ReplicationSpec struct {
	// NodeName represents which node should do replication.
	NodeName string `json:"nodeName"`
	// Tuples represents a slice of tuples.
	// +optional
	Tuples []Tuple `json:"tuples,omitempty"`
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
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

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
