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

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/inftyai/manta/api/v1alpha1"
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
	})

	type testValidatingCase struct {
		makeTorrent func() *api.Torrent
		updates     []*update
	}
	ginkgo.DescribeTable("test Torrent creation and update",
		func(tc *testValidatingCase) {
			obj := tc.makeTorrent()
			for _, update := range tc.updates {
				if update.updateFunc != nil {
					update.updateFunc(obj)
				}
				newObj := &api.Torrent{}
				gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name}, newObj)).To(gomega.Succeed())
				if update.checkFunc != nil {
					update.checkFunc(ctx, k8sClient, newObj)
				}
			}
		},
		ginkgo.Entry("Torrent with modelHub repo create", &testValidatingCase{
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b").ModelHub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, string(api.DownloadConditionType), "WaitingForDownloading", metav1.ConditionTrue)
					},
				},
			},
		}),
		ginkgo.Entry("Torrent with only modelHub file create", &testValidatingCase{
			makeTorrent: func() *api.Torrent {
				return wrapper.MakeTorrent("qwen2-7b-gguf").ModelHub("Huggingface", "Qwen/Qwen2-0.5B-Instruct-GGUF", "qwen2-0_5b-instruct-q5_k_m.gguf").Obj()
			},
			updates: []*update{
				{
					updateFunc: func(torrent *api.Torrent) {
						gomega.Expect(k8sClient.Create(ctx, torrent)).To(gomega.Succeed())
					},
					checkFunc: func(ctx context.Context, k8sClient client.Client, torrent *api.Torrent) {
						validation.ValidateTorrentStatusEqualTo(ctx, k8sClient, torrent, string(api.DownloadConditionType), "WaitingForDownloading", metav1.ConditionTrue)
					},
				},
			},
		}),
	)
})
