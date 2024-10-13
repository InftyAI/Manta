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

package main

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/inftyai/manta/agent/handler"
	api "github.com/inftyai/manta/api/v1alpha1"
)

var (
	setupLog logr.Logger
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog = ctrl.Log.WithName("setup")

	cfg, err := config.GetConfig()
	if err != nil {
		setupLog.Error(err, "failed to get config")
		os.Exit(1)
	}

	setupLog.Info("Setting up manta-agent")

	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)

	mgr, err := manager.New(cfg, manager.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "failed to initialize the manager")
		os.Exit(1)
	}

	informer, err := mgr.GetCache().GetInformer(ctx, &api.Replication{})
	if err != nil {
		setupLog.Error(err, "failed to get the informer")
		os.Exit(1)
	}

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			replication := obj.(*api.Replication)
			setupLog.Info("Add Event for Replication", "Replication", klog.KObj(replication))

			// Injected by downward API.
			nodeName := os.Getenv("NODE_NAME")
			// Filter out unrelated events.
			if nodeName != replication.Spec.NodeName || replicationReady(replication) {
				return
			}
			handler.HandleAddEvent(replication)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
		},
		DeleteFunc: func(obj interface{}) {
			replication := obj.(*api.Replication)
			setupLog.Info("Delete Event for Replication", "Replication", klog.KObj(replication))
			// TODO: delete the file by the policy.
		},
	}); err != nil {
		setupLog.Error(err, "failed to add event handlers")
		os.Exit(1)
	}

	setupLog.Info("Starting informers")
	if err := mgr.GetCache().Start(ctx); err != nil {
		setupLog.Error(err, "failed to start informers")
		os.Exit(1)
	}
}

func replicationReady(replication *api.Replication) bool {
	return apimeta.IsStatusConditionTrue(replication.Status.Conditions, api.ReadyConditionType)
}
