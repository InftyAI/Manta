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
	})

	ginkgo.It("Can download and delete a model successfully", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent)
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
	})

	ginkgo.It("Can download and delete a model with nodeSelector configured", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).NodeSelector("kubernetes.io/hostname", "kind-worker2").Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker2", 0)
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "kind-worker2")
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
		// From https://huggingface.co/facebook/opt-125m/tree/main, opt-125m has 12 files.
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker2", 12)
	})

	ginkgo.It("Sync the models successfully", func() {
		torrent := wrapper.MakeTorrent("facebook-opt-125m").Hub("Huggingface", "facebook/opt-125m", "").ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, torrent)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker", 0)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker2", 0)
			validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker3", 0)
		}()

		validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, &validation.ValidateOptions{Timeout: 5 * time.Minute})
		validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "kind-worker2")
		validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)

		// We set the replicas to 5, but we only have 3 nodes.
		newTorrent := wrapper.MakeTorrent("facebook-opt-125m-2").Hub("Huggingface", "facebook/opt-125m", "").Replicas(5).ReclaimPolicy(api.DeleteReclaimPolicy).Obj()
		gomega.Expect(k8sClient.Create(ctx, newTorrent)).To(gomega.Succeed())
		defer func() {
			gomega.Expect(k8sClient.Delete(ctx, newTorrent)).To(gomega.Succeed())
			validation.ValidateTorrentNotExist(ctx, k8sClient, newTorrent)
		}()

		// We have three nodes.
		// From https://huggingface.co/facebook/opt-125m/tree/main, opt-125m has 12 files.
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker", 12)
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker2", 12)
		validation.ValidateNodeTrackerChunkNumberEqualTo(ctx, k8sClient, "kind-worker3", 12)
	})
})
