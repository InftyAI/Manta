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

	"github.com/onsi/gomega"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/inftyai/manta/api/v1alpha1"
)

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
		if conditionType == api.DownloadConditionType {
			condition = metav1.Condition{
				Type:    api.DownloadConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "Downloading",
				Message: "Downloading chunks",
			}
		} else if conditionType == api.ReadyConditionType {
			condition = metav1.Condition{
				Type:    api.ReadyConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "Ready",
				Message: "Download chunks successfully",
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

func TorrentChunkNumber(torrent *api.Torrent) (number int) {
	if torrent.Status.Repo == nil {
		return 0
	}

	for _, obj := range torrent.Status.Repo.Objects {
		number += len(obj.Chunks)
	}
	return number
}
