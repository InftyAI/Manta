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

package controller

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/test/util"
	"github.com/inftyai/manta/test/util/validation"
	"github.com/inftyai/manta/test/util/wrapper"
)

var _ = ginkgo.Describe("Torrent controller test", func() {
	type update struct {
		updateFunc func(*api.Torrent)
		checkFunc  func(context.Context, client.Client, *api.Torrent)
	}

	ginkgo.AfterEach(func() {
		var torrents api.TorrentList
		gomega.Expect(k8sClient.List(ctx, &torrents)).To(gomega.Succeed())

		for _, torrent := range torrents.Items {
			gomega.Expect(k8sClient.Delete(ctx, &torrent)).To(gomega.Succeed())
		}

		var nodeTrackers api.NodeTrackerList
		gomega.Expect(k8sClient.List(ctx, &nodeTrackers)).To(gomega.Succeed())

		for _, nt := range nodeTrackers.Items {
			gomega.Expect(k8sClient.Delete(ctx, &nt)).To(gomega.Succeed())
		}
	})

	type testValidatingCase struct {
		precondition func() error
		makeTorrent  func() *api.Torrent
		updates      []*update
	}
	ginkgo.DescribeTable("test Torrent creation and update",
		func(tc *testValidatingCase) {
			if tc.precondition != nil {
				gomega.Expect(tc.precondition()).To(gomega.Succeed())
			}

			obj := tc.makeTorrent()
			for _, update := range tc.updates {
				if update.updateFunc != nil {
					update.updateFunc(obj)
				}
				newObj := &api.Torrent{}
				if update.checkFunc != nil {
					gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name}, newObj)).To(gomega.Succeed())
					update.checkFunc(ctx, k8sClient, newObj)
				}
			}
		},
		ginkgo.Entry("Torrent with hub repo create", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(false).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						// Preheat: false
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						gomega.Expect(len(torrent.Status.Conditions)).To(gomega.Equal(0))
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						torrent.Spec.Preheat = ptr.To[bool](true)
						gomega.Expect(k8sClient.Update(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with hub file create", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b-gguf").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with multi Replicas create", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(true).Replicas(3).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with multi Replicas and multi nodeTrackers create", &testValidatingCase{
			precondition: func() error {
				nodeTracker1 := wrapper.MakeNodeTracker("node1").Obj()
				nodeTracker2 := wrapper.MakeNodeTracker("node2").Obj()
				for _, nt := range []*api.NodeTracker{nodeTracker1, nodeTracker2} {
					if err := k8sClient.Create(ctx, nt); err != nil {
						return err
					}
				}
				return nil
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(true).Replicas(3).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						// We only have two candidates here, so only two replicas for each chunk.
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent)*2)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with nodeSelector configured", &testValidatingCase{
			precondition: func() error {
				nodeTracker1 := wrapper.MakeNodeTracker("node1").Obj()
				nodeTracker2 := wrapper.MakeNodeTracker("node2").Label("zone", "zone1").Obj()
				for _, nt := range []*api.NodeTracker{nodeTracker1, nodeTracker2} {
					if err := k8sClient.Create(ctx, nt); err != nil {
						return err
					}
				}
				return nil
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(true).Replicas(1).NodeSelector("zone", "zone1").Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node2")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with diskAware plugin enabled, sizeLimit configured", &testValidatingCase{
			precondition: func() error {
				nodeTracker1 := wrapper.MakeNodeTracker("node1").Obj()
				nodeTracker2 := wrapper.MakeNodeTracker("node2").SizeLimit("10Mi").Chunk("chunk1", 10*1024*1024).Obj()
				for _, nt := range []*api.NodeTracker{nodeTracker1, nodeTracker2} {
					if err := k8sClient.Create(ctx, nt); err != nil {
						return err
					}
				}
				return nil
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(true).Replicas(1).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with diskAware plugin enabled, score differs", &testValidatingCase{
			precondition: func() error {
				nodeTracker1 := wrapper.MakeNodeTracker("node1").Obj()
				nodeTracker2 := wrapper.MakeNodeTracker("node2").Chunk("chunk1", 1*1024*1024).Obj()
				for _, nt := range []*api.NodeTracker{nodeTracker1, nodeTracker2} {
					if err := k8sClient.Create(ctx, nt); err != nil {
						return err
					}
				}
				return nil
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b-gguf").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						// Only one chunk.
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 1)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent deletion", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").
					Preheat(true).
					Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").
					ReclaimPolicy(api.DeleteReclaimPolicy).
					Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						// The chunk name must be a real name.
						gomega.Expect(util.UpdateNodeTracker(ctx, k8sClient, "node1", "a6344aac8c09253b3b630fb776ae94478aa0275b--0001", 1024)).To(gomega.Succeed())
						gomega.Expect(k8sClient.Delete(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReclaimingConditionType, "Reclaiming", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, nil)
					},
				},
			},
		}),
		ginkgo.Entry("Recreate the same Torrent", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b-gguf").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, util.TorrentChunkNumber(torrent))
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateAllReplicationsNodeNameEqualTo(ctx, k8sClient, torrent, "node1")
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						// Recreate the same model torrent.
						newTorrent := wrapper.MakeTorrent("qwen2-7b-gguf-2").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").Obj()
						gomega.Expect(k8sClient.Create(ctx, newTorrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						// Already replicated, will be set to ready directly.
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReadyConditionType, "Ready", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Dispatch unavailable Torrent", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b-gguf").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").NodeSelector("zone", "zone1").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						// Wait for a moment to make sure it will not be ready.
						time.Sleep(3 * time.Second)
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.PendingConditionType, "Pending", metav1.ConditionTrue, nil)
						validation.ValidateReplicationsNumberEqualTo(ctx, k8sClient, torrent, 0)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with ttl is zero", &testValidatingCase{
			precondition: func() error {
				nodeTracker := wrapper.MakeNodeTracker("node1").Obj()
				return k8sClient.Create(ctx, nodeTracker)
			},
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").Preheat(true).TTL(0).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReplicateConditionType)
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, api.ReplicateConditionType, "Replicating", metav1.ConditionTrue, nil)
					},
				},
				{
					updateFunc: func(torrent *api.Torrent) {
						util.UpdateReplicationsCondition(ctx, k8sClient, torrent, api.ReadyConditionType)
						// Once ready, Torrent will be deleted immediately.
						validation.ValidateTorrentNotExist(ctx, k8sClient, torrent.Name, nil)
					},
				},
			},
		}),
	)
})
