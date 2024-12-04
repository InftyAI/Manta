/*
Copyright 2024 The Kubernetes Authors.
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

package e2e

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/test/util"
	"github.com/inftyai/manta/test/util/validation"
	"github.com/inftyai/manta/test/util/wrapper"
)

var _ = ginkgo.Describe("torrent e2e test", func() {
	// Each test runs in a separate namespace.
	var ns *corev1.Namespace

	ginkgo.BeforeEach(func() {
		// Create test namespace before each test.
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-ns-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})
	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.Delete(ctx, ns)).To(gomega.Succeed())

		var torrents api.TorrentList
		gomega.Expect(k8sClient.List(ctx, &torrents)).To(gomega.Succeed())

		for _, torrent := range torrents.Items {
			gomega.Expect(k8sClient.Delete(ctx, &torrent)).To(gomega.Succeed())
		}
	})

	ginkgo.It("Can download and delete a model successfully", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, nil)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 0, "kind-worker", "kind-worker2", "kind-worker3")
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
	})

	ginkgo.It("Can download and delete a model with nodeSelector configured", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).NodeSelector("kubernetes.io/hostname", "kind-worker2").Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, nil)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 0, "kind-worker", "kind-worker2", "kind-worker3")
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "kind-worker2")
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
		// From https://huggingface.co/facebook/opt-125m/tree/main, opt-125m has 12 files.
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 12, "kind-worker2")
	})

	ginkgo.It("Sync the models successfully", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, nil)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 0, "kind-worker", "kind-worker2", "kind-worker3")
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)

		// We set the replicas to 5, but we only have 3 nodes.
		newTorrent := wrapper.MakeTorrent("facebook-opt-125m-2").Hub("Huggingface", "facebook/opt-125m", "").Replicas(5).ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, newTorrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, newTorrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, newTorrent.Name, nil)
		}()

		// We have three nodes.
		// From https://huggingface.co/facebook/opt-125m/tree/main, opt-125m has 12 files.
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 12, "kind-worker", "kind-worker2", "kind-worker3")
	})

	ginkgo.It("Torrent will be auto GCed with TTLSecondsAfterReady", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").TTL(0).Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())

		validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 0, "kind-worker", "kind-worker2", "kind-worker3")
	})

	ginkgo.It("Pod with Torrent label will download models", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Preheat(false).Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		pod := wrapper.MakePod("test-pod", ns.Name).Label(api.TorrentNameLabelKey, torrent.Name).Obj()
		gomega.Expect(k8sClient.Create(ctx, pod)).To(gomega.Succeed())

		// Wait for pod scheduled and forked Torrent auto GCed.
		util.PodScheduled(ctx, k8sClient, pod)
		validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name+"--tmp--"+pod.Spec.NodeName, &validation.ValidateOptions{Timeout: 5 * time.Minute})

		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, 12, pod.Spec.NodeName)
	})
})
