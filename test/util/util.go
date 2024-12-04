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

import (
	"context"
	"fmt"
	"os"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/inftyai/manta/api/v1alpha1"
)

func UpdateNodeTracker(ctx context.Context, k8sClient client.Client, ntName string, chunkName string, size int32) error {
	nt := &api.NodeTracker{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: ntName}, nt); err != nil {
		return err
	}
	nt.Spec.Chunks = append(nt.Spec.Chunks, api.ChunkTracker{ChunkName: chunkName, SizeBytes: int64(size)})
	if err := k8sClient.Update(ctx, nt); err != nil {
		return err
	}
	return nil
}

func UpdateReplicationsCondition(ctx context.Context, k8sClient client.Client, torrent *api.Torrent, conditionType string) {
	gomega.Eventually(func() error {
		replicationList := &api.ReplicationList{}
		selector := labels.SelectorFromSet(labels.Set{api.TorrentNameLabelKey: torrent.Name})
		if err := k8sClient.List(ctx, replicationList, &client.ListOptions{
			LabelSelector: selector,
		}); err != nil {
			return err
		}

		condition := metav1.Condition{}
		if conditionType == api.ReplicateConditionType {
			condition = metav1.Condition{
				Type:    api.ReplicateConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "Replicating",
				Message: "Replicating chunks",
			}
		} else if conditionType == api.ReadyConditionType {
			condition = metav1.Condition{
				Type:    api.ReadyConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "Ready",
				Message: "Chunks replicated successfully",
			}
		}

		for _, replication := range replicationList.Items {
			apimeta.SetStatusCondition(&replication.Status.Conditions, condition)
			replication.Status.Phase = &conditionType

			if err := k8sClient.Status().Update(ctx, &replication); err != nil {
				return err
			}
		}

		return nil
	}, Timeout, Interval).Should(gomega.Succeed())
}

func PodScheduled(ctx context.Context, k8sClient client.Client, pod *corev1.Pod) {
	gomega.Eventually(func() bool {
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, pod); err != nil {
			return false
		}
		return pod.Spec.NodeName != ""
	}, Timeout, Interval).Should(gomega.BeTrue())
}

func TorrentChunkNumber(torrent *api.Torrent) (number int) {
	if torrent.Status.Repo == nil {
		return 0
	}

	for _, obj := range torrent.Status.Repo.Objects {
		number += len(obj.Chunks)
	}
	return number
}

func Apply(ctx context.Context, k8sClient client.Client, path string, ns string, action string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := applyYaml(ctx, k8sClient, path+"/"+entry.Name(), ns, action); err != nil {
			return err
		}
	}
	return nil
}

func applyYaml(ctx context.Context, k8sClient client.Client, file string, ns string, action string) error {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %v", err)
	}

	decode := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, _, err = decode.Decode(yamlFile, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode YAML into Unstructured object: %v", err)
	}

	if ns != "" {
		obj.SetNamespace(ns)
	}

	if action == "create" {
		if err = k8sClient.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create resource: %v", err)
		}
		return nil
	}

	if action == "delete" {
		if err = k8sClient.Delete(ctx, obj); err != nil {
			return fmt.Errorf("failed to delete resource: %v", err)
		}
		return nil
	}

	return nil
}
