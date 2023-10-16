// Copyright 2023 The Skaffold Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	SetupCoverageSignalHandler(syscall.SIGUSR1)
	http.HandleFunc("/", hello)
	port := "8080"
	if portEnv, exists := os.LookupEnv("PORT"); exists {
		port = portEnv
	}
	log.Printf("Listening on port %s\n", port)
	return http.ListenAndServe(":"+port, nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got request: %s %s", r.Method, r.URL.Path)
	fmt.Fprintln(w, "Hello, World!")
}
