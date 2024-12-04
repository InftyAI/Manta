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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/inftyai/manta/test/util/wrapper"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/inftyai/manta/api/v1alpha1"
	defaults "github.com/inftyai/manta/pkg"
)

var _ = ginkgo.Describe("Pod default and validation", func() {

	var ns *corev1.Namespace

	ginkgo.BeforeEach(func() {
		// Create test namespace before each test.
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-ns-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
		gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ns.Name}, ns)).Should(gomega.Succeed())
	})

	type testDefaultingCase struct {
		creationFunc func() *corev1.Pod
		wantPod      func() *corev1.Pod
	}
	ginkgo.DescribeTable("test validating",
		func(tc *testDefaultingCase) {
			pod := tc.creationFunc()
			gomega.Expect(k8sClient.Create(ctx, pod)).To(gomega.Succeed())
			// only diff the initContainer.
			gomega.Expect(cmp.Diff(pod.Spec.InitContainers, tc.wantPod().Spec.InitContainers,
				cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePolicy", "TerminationMessagePath"))).To(gomega.BeEmpty())
		},
		ginkgo.Entry("Pod not managed by manta", &testDefaultingCase{
			creationFunc: func() *corev1.Pod {
				return wrapper.MakePod("pod1", ns.Name).Obj()
			},
			wantPod: func() *corev1.Pod {
				return wrapper.MakePod("pod1", ns.Name).Obj()
			},
		}),
		ginkgo.Entry("Pod managed by manta", &testDefaultingCase{
			creationFunc: func() *corev1.Pod {
				return wrapper.MakePod("pod1", ns.Name).Label(api.TorrentNameLabelKey, "torrent").Obj()
			},
			wantPod: func() *corev1.Pod {
				return wrapper.MakePod("pod1", ns.Name).Label(api.TorrentNameLabelKey, "torrent").
					InitContainer("preheat").InitContainerImage("preheat", defaults.PREHEAT_IMAGE).InitContainerImagePolicy("preheat", "IfNotPresent").
					InitContainerCommands("preheat", "/manager").InitContainerPort("preheat", "http", 9090, "TCP").
					Obj()
			},
		}),
	)
})
