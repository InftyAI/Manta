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

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/inftyai/manta/agent/pkg/handler"
	"github.com/inftyai/manta/api"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/sync", handler.SendChunk)
	server := &http.Server{Addr: ":" + api.HttpPort, Handler: mux}

	go func() {
		logger := log.FromContext(ctx)
		logger.Info("Server started on port 9090")

		if err := server.ListenAndServe(); err != nil {
			logger.Error(err, "listen and server error")
			cancel()
		}
	}()

	<-ctx.Done()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	logger := log.FromContext(ctx)

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Error(err, "server shutdown error")
	}

	logger.Info("server shutdown successfully")
}
