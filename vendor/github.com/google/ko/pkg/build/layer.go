// Copyright 2020 ko Build Authors All Rights Reserved.
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

package build

import (
	"io"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type lazyLayer struct {
	diffid v1.Hash
	desc   v1.Descriptor

	sync.Once
	buildLayer func() (v1.Layer, error)
	layer      v1.Layer
	err        error
}

// All this info is cached by previous builds.
func (l *lazyLayer) Digest() (v1.Hash, error) {
	return l.desc.Digest, nil
}

func (l *lazyLayer) DiffID() (v1.Hash, error) {
	return l.diffid, nil
}

func (l *lazyLayer) Size() (int64, error) {
	return l.desc.Size, nil
}

func (l *lazyLayer) MediaType() (types.MediaType, error) {
	return l.desc.MediaType, nil
}

// This is only called if the registry doesn't have this blob already.
func (l *lazyLayer) Compressed() (io.ReadCloser, error) {
	layer, err := l.compute()
	if err != nil {
		return nil, err
	}
	return layer.Compressed()
}

// This should never actually be called but we need it to impl v1.Layer.
func (l *lazyLayer) Uncompressed() (io.ReadCloser, error) {
	layer, err := l.compute()
	if err != nil {
		return nil, err
	}
	return layer.Uncompressed()
}

func (l *lazyLayer) compute() (v1.Layer, error) {
	l.Once.Do(func() {
		l.layer, l.err = l.buildLayer()
	})
	return l.layer, l.err
}
