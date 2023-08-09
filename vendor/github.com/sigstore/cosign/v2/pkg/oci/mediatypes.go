//
// Copyright 2021 The Sigstore Authors.
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

import (
	"strconv"

	"github.com/sigstore/cosign/v2/pkg/cosign/env"
)

const (
	// Deprecated: use `pkg/cosign/env/VariableDockerMediaTypes` instead.
	DockerMediaTypesEnv = env.VariableDockerMediaTypes
)

func DockerMediaTypes() bool {
	if b, err := strconv.ParseBool(env.Getenv(env.VariableDockerMediaTypes)); err == nil {
		return b
	}
	return false
}
