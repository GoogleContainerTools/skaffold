/*
Copyright 2018 The Skaffold Authors

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

package kaniko

import (
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// Builder builds docker artifacts on Kubernetes, using Kaniko.
type Builder struct {
	*latest.KanikoBuild

	timeout time.Duration
}

// NewBuilder creates a new Builder that builds artifacts with Kaniko.
func NewBuilder(cfg *latest.KanikoBuild) (*Builder, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "parsing timeout")
	}

	return &Builder{
		KanikoBuild: cfg,
		timeout:     timeout,
	}, nil
}

// Labels are labels specific to Kaniko builder.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "kaniko",
	}
}
