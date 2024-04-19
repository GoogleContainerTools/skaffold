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

import "fmt"

// MaxLayerSizeExceeded is an error indicating that the layer is too big to read into memory and cosign should abort processing it.
type MaxLayerSizeExceeded struct {
	value   uint64
	maximum uint64
}

func NewMaxLayerSizeExceeded(value, maximum uint64) *MaxLayerSizeExceeded {
	return &MaxLayerSizeExceeded{value, maximum}
}

func (e *MaxLayerSizeExceeded) Error() string {
	return fmt.Sprintf("size of layer (%d) exceeded the limit (%d)", e.value, e.maximum)
}
