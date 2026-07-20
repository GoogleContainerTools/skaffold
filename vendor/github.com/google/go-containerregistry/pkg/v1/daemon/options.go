// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	"context"
	"io"

	"github.com/moby/moby/client"
)

// ImageOption is an alias for Option.
// Deprecated: Use Option instead.
type ImageOption Option

// Option is a functional option for daemon operations.
type Option func(*options)

// bufferMode controls how the image tarball is buffered.
type bufferMode int

const (
	bufferMemory bufferMode = iota // default: buffer entire image in memory
	bufferNone                     // no buffering: re-save on each access
	bufferFile                     // buffer to a temp file on disk
)

type options struct {
	ctx        context.Context
	client     Client
	bufferMode bufferMode
}

var defaultClient = func() (Client, error) {
	return client.New(client.FromEnv)
}

func makeOptions(opts ...Option) (*options, error) {
	o := &options{
		bufferMode: bufferMemory,
		ctx:        context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}

	if o.client == nil {
		apiClient, err := defaultClient()
		if err != nil {
			return nil, err
		}
		o.client = apiClient
	}
	_, _ = o.client.Ping(o.ctx, client.PingOptions{
		NegotiateAPIVersion: true,
	})

	return o, nil
}

// WithBufferedOpener buffers the entire image into memory.
func WithBufferedOpener() Option {
	return func(o *options) {
		o.bufferMode = bufferMemory
	}
}

// WithUnbufferedOpener streams the image to avoid buffering it.
// Each access triggers a new image save.
func WithUnbufferedOpener() Option {
	return func(o *options) {
		o.bufferMode = bufferNone
	}
}

// WithFileBufferedOpener buffers the image to a temporary file on disk.
// This avoids holding the entire image in memory while still only
// performing a single image save. The temporary file is cleaned up via
// runtime.AddCleanup on the imageOpener.
func WithFileBufferedOpener() Option {
	return func(o *options) {
		o.bufferMode = bufferFile
	}
}

// WithClient is a functional option to allow injecting a docker client.
//
// By default, github.com/docker/docker/client.FromEnv is used.
func WithClient(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithContext is a functional option to pass through a context.Context.
//
// By default, context.Background() is used.
func WithContext(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}

// Client represents the subset of a docker client that the daemon
// package uses.
type Client interface {
	Ping(ctx context.Context, options client.PingOptions) (client.PingResult, error)
	ImageSave(ctx context.Context, images []string, _ ...client.ImageSaveOption) (client.ImageSaveResult, error)
	ImageLoad(ctx context.Context, input io.Reader, _ ...client.ImageLoadOption) (client.ImageLoadResult, error)
	ImageTag(ctx context.Context, options client.ImageTagOptions) (client.ImageTagResult, error)
	ImageInspect(ctx context.Context, image string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error)
	ImageHistory(ctx context.Context, image string, _ ...client.ImageHistoryOption) (client.ImageHistoryResult, error)
}
