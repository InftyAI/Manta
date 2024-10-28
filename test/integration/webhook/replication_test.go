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

// TODO: once source.hub != nil, it's address must not be nil.
// TODO: add validation to URI, must be <host>://<index>
var _ = ginkgo.Describe("Replication default and validation", func() {

	// Delete all the Replications for each case.
	ginkgo.AfterEach(func() {
		var replications api.ReplicationList
		gomega.Expect(k8sClient.List(ctx, &replications)).To(gomega.Succeed())

		for _, torrent := range replications.Items {
			gomega.Expect(k8sClient.Delete(ctx, &torrent)).To(gomega.Succeed())
		}
	})

	type testValidatingCase struct {
		replication func() *api.Replication
		failed      bool
	}
	ginkgo.DescribeTable("test validating",
		func(tc *testValidatingCase) {
			if tc.failed {
				gomega.Expect(k8sClient.Create(ctx, tc.replication())).Should(gomega.HaveOccurred())
			} else {
				gomega.Expect(k8sClient.Create(ctx, tc.replication())).To(gomega.Succeed())
			}
		},
		ginkgo.Entry("replication with hub set", &testValidatingCase{
			replication: func() *api.Replication {
				return wrapper.MakeReplication("fake-replication").SourceOfHub("Huggingface", "Qwen/Qwen2-7B-Instruct", "", "").DestinationOfURI("localhost://destination").Obj()
			},
			failed: false,
		}),
		ginkgo.Entry("replication with hub and URI unset", &testValidatingCase{
			replication: func() *api.Replication {
				replication := wrapper.MakeReplication("fake-replication").Obj()
				return replication
			},
			failed: true,
		}),
		ginkgo.Entry("once source is localhost, destination must be nil", &testValidatingCase{
			replication: func() *api.Replication {
				replication := wrapper.MakeReplication("fake-replication").SourceOfURI("localhost://source").DestinationOfURI("localhost://destination").Obj()
				return replication
			},
			failed: true,
		}),
		ginkgo.Entry("destination.uri must be localhost once source.hub is not nil", &testValidatingCase{
			replication: func() *api.Replication {
				return wrapper.MakeReplication("fake-replication").SourceOfHub("Huggingface", "Qwen/Qwen2-7B-Instruct", "", "").DestinationOfURI("remote://destination").Obj()
			},
			failed: true,
		}),
	)
})
