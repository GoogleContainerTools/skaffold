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
	"log"
	"os"
	"os/signal"
	"runtime/coverage"
	"syscall"
)

var onlyOneCoverageSignalHandler = make(chan struct{})

// SetupCoverageSignalHandler creates a channel and relays the provided signals
// to this channel. It also starts a goroutine that receives on that channel.
// When the goroutine receives a signal, it writes profile data files to the
// directory specified by the `GOCOVERDIR` environment variable. After writing
// the data, the goroutine clears the coverage counters.
//
// Clearing the counters is only possible if the binary was built with
// `-covermode=atomic`.
//
// If no signals are provided as arguments, the function defaults to relaying
// `SIGUSR1`.
//
// If the `GOCOVERDIR` environment variable is _not_ set, this function does
// nothing.
//
// References:
// - https://go.dev/testing/coverage/
// - https://pkg.go.dev/runtime/coverage
func SetupCoverageSignalHandler(signals ...os.Signal) {
	close(onlyOneCoverageSignalHandler) // panics when called twice

	// Default to USR1 signal if no signals provided in the function argument.
	if len(signals) < 1 {
		signals = []os.Signal{syscall.SIGUSR1}
	}

	// Set up the signal handler only if GOCOVERDIR is set.
	coverDir, exists := os.LookupEnv("GOCOVERDIR")
	if !exists {
		return
	}

	log.Printf("Configuring coverage profile data signal handler, listening for %v", signals)
	c := make(chan os.Signal)
	signal.Notify(c, signals...)
	go func() {
		for {
			signal := <-c
			log.Printf("Got %v, writing coverage profile data files to %q", signal, coverDir)
			if err := coverage.WriteCountersDir(coverDir); err != nil {
				log.Printf("Could not write coverage profile data files to the directory %q: %+v", coverDir, err)
			}
			if err := coverage.ClearCounters(); err != nil {
				log.Printf("Could not reset coverage counter variables: %+v", err)
			}
		}
	}()
}
