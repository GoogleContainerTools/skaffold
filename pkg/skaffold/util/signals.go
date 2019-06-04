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

package util

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func CancelWithCtrlC(ctx context.Context, action func(context.Context, io.Writer) error) func(io.Writer) error {
	return func(out io.Writer) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go CatchCtrlC(cancel)
		return action(ctx, out)
	}
}

func CatchCtrlC(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		cancel()
	}()
}

func WaitForSignalOrCtrlC(ctx context.Context, trigger chan bool) {
	proceed := make(chan bool, 1)
	go func() {
		<-trigger
		proceed <- true
	}()
	go func(ctx context.Context) {
		_, cancel := context.WithCancel(ctx)
		CatchCtrlC(cancel)
		proceed <- true
	}(ctx)
	<-proceed
}
