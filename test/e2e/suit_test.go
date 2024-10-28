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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api "github.com/inftyai/manta/api/v1alpha1"
	"github.com/inftyai/manta/test/util"
	"github.com/inftyai/manta/test/util/wrapper"
)

var cfg *rest.Config
var k8sClient client.Client
var ctx context.Context
var cancel context.CancelFunc

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	// cfg is defined in this file globally.
	cfg = config.GetConfigOrDie()
	Expect(cfg).NotTo(BeNil())

	err := api.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = admissionv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	readyForTesting(k8sClient)
})

var _ = AfterSuite(func() {
	cancel()
})

func readyForTesting(client client.Client) {
	By("waiting for webhooks to ready")

	// To verify that webhooks are ready, let's create a simple Replication.
	replication := wrapper.MakeReplication("sample-replication").
		NodeName("unknown-node").
		ChunkName("chunk1").
		SizeBytes(1024).
		SourceOfURI("localhost:///workspace/models/modelA").
		Obj()

	// Once the creation succeeds, that means the webhooks are ready
	// and we can begin testing.
	Eventually(func() error {
		return client.Create(ctx, replication)
	}, util.Timeout, util.Interval).Should(Succeed())

	// Delete this replication before beginning tests.
	Expect(client.Delete(ctx, replication)).To(Succeed())
	Eventually(func() error {
		return client.Get(ctx, types.NamespacedName{Name: replication.Name}, &api.Replication{})
	}).ShouldNot(Succeed())

	By("waiting for nodeTrackers to ready")
	// Hardcoded the namespace here because we just can't change the namespace dynamically.
	Expect(util.Apply(ctx, k8sClient, "../../agent/deploy", "manta-system", "create")).To(Succeed())
	Eventually(func() error {
		nodeTrackers := &api.NodeTrackerList{}
		if err := client.List(ctx, nodeTrackers); err != nil {
			return err
		}
		if len(nodeTrackers.Items) == 0 {
			return fmt.Errorf("no nodeTrackers")
		}
		return nil
	}, util.Timeout, util.Interval).Should(Succeed())
}
