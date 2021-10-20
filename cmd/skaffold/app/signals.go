/*
Copyright 2019 The Skaffold Authors

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

package app

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func catchCtrlC(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		signal.Stop(signals)
		cancel()
	}()
}

func catchSIGUSR1() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGUSR1,
	)

	go func() {
		for {
			<-signals
			buf := make([]byte, 1<<20)
			runtime.Stack(buf, true)
			os.Stderr.Write([]byte("Dumping stack traces:"))
			os.Stderr.Write(buf)
			os.Stderr.Write([]byte("Done dumping stack traces"))
		}
	}()
}
