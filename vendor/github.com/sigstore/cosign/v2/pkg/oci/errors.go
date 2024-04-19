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

package oci

import "fmt"

// MaxLayersExceeded is an error indicating that the artifact has too many layers and cosign should abort processing it.
type MaxLayersExceeded struct {
	value   int64
	maximum int64
}

func NewMaxLayersExceeded(value, maximum int64) *MaxLayersExceeded {
	return &MaxLayersExceeded{value, maximum}
}

func (e *MaxLayersExceeded) Error() string {
	return fmt.Sprintf("number of layers (%d) exceeded the limit (%d)", e.value, e.maximum)
}
