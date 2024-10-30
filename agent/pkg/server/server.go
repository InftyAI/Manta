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
	"fmt"
	"net/http"
	"time"

	"github.com/inftyai/manta/agent/pkg/handler"
)

func Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/sync", handler.SendChunk)
	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		fmt.Println("Server started on port 8080")

		if err := server.ListenAndServe(); err != nil {
			fmt.Printf("ListenAndServe error: %s\n", err)
			cancel()
		}
	}()

	<-ctx.Done()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		fmt.Printf("Server shutdown error: %s\n", err)
	}

	fmt.Println("Server shutdown successfully")
}
