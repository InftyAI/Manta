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

// This is inspired by https://github.com/InftyAI/llmaz.
// ModelHub represents the model registry for model downloads.
type ModelHub struct {
	// Name refers to the model registry, such as huggingface.
	// +kubebuilder:default=Huggingface
	// +kubebuilder:validation:Enum={Huggingface,ModelScope}
	// +optional
	Name *string `json:"name,omitempty"`
	// ModelID refers to the model identifier on model hub,
	// such as meta-llama/Meta-Llama-3-8B.
	ModelID string `json:"modelID"`
	// Filename refers to a specified model file rather than the whole repo.
	// This is helpful to download a specified GGUF model rather than downloading
	// the whole repo which includes all kinds of quantized models.
	// TODO: this is only supported with Huggingface, add support for ModelScope
	// in the near future.
	Filename *string `json:"filename,omitempty"`
	// Revision refers to a Git revision id which can be a branch name, a tag, or a commit hash.
	// Most of the time, you don't need to specify it.
	// +optional
	Revision *string `json:"revision,omitempty"`
}

// URIProtocol represents the protocol of the URI.
type URIProtocol string

// TorrentSpec defines the desired state of Torrent
type TorrentSpec struct {
	// ModelHub represents the model registry for model downloads.
	// ModelHub and URI are exclusive.
	// +optional
	ModelHub *ModelHub `json:"modelHub,omitempty"`

	// TODO: not implemented.
	// URI represents a various kinds of file sources following the uri protocol, e.g.
	// 	- Image: img://nginx:1.14.2
	// 	- OSS: oss://<bucket>.<endpoint>/<path-to-your-files>
	// +optional
	// URI *URIProtocol `json:"uriuri,omitempty"`

	// Replicas represents the replication number of each file.
	// +kubebuilder:default=1
	// +optional
	Replicas *int32 ` json:"replicas,omitempty"`
}

type TrackerState string

const (
	DownloadTrackerState TrackerState = "Downloading"
	ReadyTrackerState    TrackerState = "Ready"
)

type FileTracker struct {
	// Name represents the name of the file.
	Name string `json:"name"`
	// State represents the state of the file, whether Pending
	// for download or downloaded ready.
	State TrackerState `json:"State"`
	// SizeBytes represents the file size.
	SizeBytes int64 `json:"sizeBytes"`
}

const (
	// DownloadConditionType represents the Torrent is under downloading.
	DownloadConditionType = string(DownloadTrackerState)
	// ReadyConditionType represents the Torrent is downloaded successfully.
	ReadyConditionType = string(ReadyTrackerState)
)

// TorrentStatus defines the observed state of Torrent
type TorrentStatus struct {
	// Conditions represents the Torrent condition.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Files tracks the files belong to the source.
	Files []FileTracker `json:"files,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Torrent is the Schema for the torrents API
type Torrent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TorrentSpec   `json:"spec,omitempty"`
	Status TorrentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TorrentList contains a list of Torrent
type TorrentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Torrent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Torrent{}, &TorrentList{})
}
