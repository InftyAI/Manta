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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TorrentNameLabelKey        = "manta.io/torrent-name"
	TorrentProtectionFinalizer = "manta.io/torrent-protect"
	ParentPodNameAnnoKey       = "manta.io/parent-pod-name"

	HUGGINGFACE_MODEL_HUB = "Huggingface"
)

// This is inspired by https://github.com/InftyAI/llmaz.
// Hub represents the model registry for model downloads.
type Hub struct {
	// TODO: support ModelScope
	// Name refers to the model registry, such as huggingface.
	// +kubebuilder:default=Huggingface
	// +kubebuilder:validation:Enum={Huggingface}
	// +optional
	Name *string `json:"name,omitempty"`
	// RepoID refers to the identifier on hub,
	// such as meta-llama/Meta-Llama-3-8B.
	RepoID string `json:"repoID"`
	// Filename refers to a specified model file rather than the whole repo.
	// This is helpful to download a specified GGUF model rather than downloading
	// the whole repo which includes all kinds of quantized models.
	// TODO: this is only supported with Huggingface, add support for ModelScope
	// in the near future.
	Filename *string `json:"filename,omitempty"`
	// Revision refers to a Git revision id which can be a branch name, a tag, or a commit hash.
	// +kubebuilder:default=main
	// +optional
	Revision *string `json:"revision,omitempty"`
}

// URIProtocol represents the protocol of the URI.
type URIProtocol string

type ReclaimPolicy string

const (
	// RetainReclaimPolicy represents keep the files when Torrent is deleted.
	RetainReclaimPolicy ReclaimPolicy = "Retain"
	// DeleteReclaimPolicy represents delete the files when Torrent is deleted.
	DeleteReclaimPolicy ReclaimPolicy = "Delete"
)

// TorrentSpec defines the desired state of Torrent
type TorrentSpec struct {
	// Preheat represents whether we should preload the model.
	// Preheat can only be transitioned from false to true, not the other way around.
	// +kubebuilder:default=true
	// +optional
	Preheat *bool `json:"preheat,omitempty"`

	// Hub represents the model registry for model downloads.
	// Hub and URI are exclusive.
	// +optional
	Hub *Hub `json:"hub,omitempty"`

	// TODO: not implemented.
	// URI represents a various kinds of file sources following the uri protocol, e.g.
	// 	- Image: img://nginx:1.14.2
	// 	- OSS: oss://<bucket>.<endpoint>/<path-to-your-files>
	// +optional
	// URI *URIProtocol `json:"uri,omitempty"`

	// Replicas represents the replication number of each object.
	// +kubebuilder:default=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// ReclaimPolicy represents how to handle the file replicas when Torrent is deleted.
	// Be careful to use the Delete policy because once two Torrents refer to the same
	// repo, delete one Torrent will remove the whole files.
	// +kubebuilder:default=Retain
	// +kubebuilder:validation:Enum={Retain,Delete}
	// +optional
	ReclaimPolicy *ReclaimPolicy `json:"reclaimPolicy,omitempty"`
	// TTLSecondsAfterReady represents the waiting time to delete the Torrent once Ready.
	// Default to nil indicates Torrent will not be deleted.
	// TODO: We only support nil and 0 right now.
	// +optional
	TTLSecondsAfterReady *time.Duration `json:"ttlSecondsAfterReady,omitempty"`
	// NodeSelector represents the node constraints to download the chunks.
	// It can be used to download the model to a specified node for preheating.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type TrackerState string

const (
	// Pending means the chunk is waiting for downloading.
	PendingTrackerState TrackerState = "Pending"
	// Ready means the chunk is ready for downloading or downloaded.
	ReadyTrackerState TrackerState = "Ready"
	// Deleting means the chunk is being removed.
	DeletingTrackerState TrackerState = "Deleting"
)

type ChunkStatus struct {
	// Name represents the name of the chunk.
	// The chunk name is formatted as: <object hash>--<chunk number>,
	// e.g. "945c19bff66ba533eb2032a33dcc6281c4a1e032--0210", which means:
	// - the object hash is 945c19bff66ba533eb2032a33dcc6281c4a1e032
	// - the chunk is the second chunk of the total 10 chunks
	Name string `json:"name"`
	// SizeBytes represents the chunk size.
	SizeBytes int64 `json:"sizeBytes"`
	// State represents the state of the chunk, whether in Pending or Ready.
	// Chunks in Pending state will lead to Replication creation operations.
	State TrackerState `json:"state"`
}

type ObjectType string

const (
	FileObjectType      ObjectType = "file"
	DirectoryObjectType ObjectType = "directory"
)

// ObjectStatus tracks the object info.
type ObjectStatus struct {
	// Path represents the path of the object.
	Path string `json:"path"`
	// Chunks represents the whole chunks which makes up the object.
	// +optional
	Chunks []ChunkStatus `json:"chunks,omitempty"`
	// Type represents the object type, limits to file or directory.
	// +kubebuilder:validation:Enum={file,directory}
	Type ObjectType `json:"type"`
}

type RepoStatus struct {
	// Objects represents the whole objects belongs to the repo.
	// +optional
	Objects []ObjectStatus `json:"objects,omitempty"`
}

const (
	// PendingConditionType represents the Torrent is Pending.
	PendingConditionType = "Pending"
	// ReplicateConditionType represents the Torrent is under replicating, downloading or syncing.
	ReplicateConditionType = "Replicating"
	// ReadyConditionType represents the Torrent is replicated successfully.
	ReadyConditionType = "Ready"
	// ReclaimingConditionType represents the Torrent is removing chunks.
	ReclaimingConditionType = "Reclaiming"
)

// TorrentStatus defines the observed state of Torrent
type TorrentStatus struct {
	// Conditions represents the Torrent condition.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Repo tracks the objects belong to the source.
	Repo *RepoStatus `json:"repo,omitempty"`
	// Phase represents the current state.
	// +optional
	Phase *string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

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
