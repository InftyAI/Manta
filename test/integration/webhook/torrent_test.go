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

package webhook

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/test/util/wrapper"
)

var _ = ginkgo.Describe("Torrent default and validation", func() {

	// Delete all the Torrents for each case.
	ginkgo.AfterEach(func() {
		var torrents api.TorrentList
		gomega.Expect(k8sClient.List(ctx, &torrents)).To(gomega.Succeed())

		for _, torrent := range torrents.Items {
			gomega.Expect(k8sClient.Delete(ctx, &torrent)).To(gomega.Succeed())
		}
	})

	type testValidatingCase struct {
		creationFunc func() *api.Torrent
		createFailed bool
		updateFunc   func(*api.Torrent) *api.Torrent
		updateFiled  bool
	}
	ginkgo.DescribeTable("test validating",
		func(tc *testValidatingCase) {
			torrent := tc.creationFunc()
			err := k8sClient.Create(ctx, torrent)

			if tc.createFailed {
				gomega.Expect(err).To(gomega.HaveOccurred())
				return
			} else {
				gomega.Expect(err).To(gomega.Succeed())
			}

			gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: torrent.Name, Namespace: torrent.Namespace}, torrent)).Should(gomega.Succeed())

			if tc.updateFunc != nil {
				err = k8sClient.Update(ctx, tc.updateFunc(torrent))
				if tc.updateFiled {
					gomega.Expect(err).To(gomega.HaveOccurred())
				} else {
					gomega.Expect(err).To(gomega.Succeed())
				}
			}
		},
		ginkgo.Entry("torrent hub set", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: false,
		}),
		ginkgo.Entry("torrent hub not set", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Obj()
			},
			createFailed: true,
		}),
		ginkgo.Entry("unknown hub not supported", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Hub("ModelScope", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: true,
		}),
		ginkgo.Entry("preheat from false to true should be succeeded", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: false,
			updateFunc: func(torrent *api.Torrent) *api.Torrent {
				torrent.Spec.Preheat = ptr.To[bool](true)
				return torrent
			},
			updateFiled: false,
		}),
		ginkgo.Entry("preheat from true to false should be failed", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Preheat(true).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: false,
			updateFunc: func(torrent *api.Torrent) *api.Torrent {
				torrent.Spec.Preheat = ptr.To[bool](false)
				return torrent
			},
			updateFiled: true,
		}),
		ginkgo.Entry("ttlSecondsAfterReady is 0", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Preheat(true).TTL(0).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: false,
		}),
		ginkgo.Entry("ttlSecondsAfterReady not nil or 0", &testValidatingCase{
			creationFunc: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Preheat(true).TTL(1).Hub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			createFailed: true,
		}),
	)
})
