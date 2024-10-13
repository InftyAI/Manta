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
		torrent func() *api.Torrent
		failed  bool
	}
	ginkgo.DescribeTable("test validating",
		func(tc *testValidatingCase) {
			if tc.failed {
				gomega.Expect(k8sClient.Create(ctx, tc.torrent())).Should(gomega.HaveOccurred())
			} else {
				gomega.Expect(k8sClient.Create(ctx, tc.torrent())).To(gomega.Succeed())
			}
		},
		ginkgo.Entry("torrent modelHub set", &testValidatingCase{
			torrent: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").ModelHub("Huggingface", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			failed: false,
		}),
		ginkgo.Entry("torrent modelHub not set", &testValidatingCase{
			torrent: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").Obj()
			},
			failed: true,
		}),
		ginkgo.Entry("unknown modelHub not supported", &testValidatingCase{
			torrent: func() *api.Torrent {
				return wrapper.MakeTorrent("download-qwen").ModelHub("ModelScope", "Qwen/Qwen2-7B-Instruct", "").Obj()
			},
			failed: true,
		}),
	)
})
