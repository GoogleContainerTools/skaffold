// Copyright 2024 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package payload

import (
	"github.com/dustin/go-humanize"
	"github.com/sigstore/cosign/v2/pkg/cosign/env"
)

const defaultMaxSize = uint64(134217728) // 128MiB

func CheckSize(size uint64) error {
	maxSize := defaultMaxSize
	maxSizeOverride, exists := env.LookupEnv(env.VariableMaxAttachmentSize)
	if exists {
		var err error
		maxSize, err = humanize.ParseBytes(maxSizeOverride)
		if err != nil {
			maxSize = defaultMaxSize
		}
	}
	if size > maxSize {
		return NewMaxLayerSizeExceeded(size, maxSize)
	}
	return nil
}
