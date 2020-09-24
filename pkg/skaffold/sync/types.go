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

package sync

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
)

type syncMap map[string][]string

type Item struct {
	Image  string
	Copy   map[string][]string
	Delete map[string][]string
}

type Syncer interface {
	Sync(context.Context, *Item) error
}

type podSyncer struct {
	kubectl    *pkgkubectl.CLI
	namespaces []string
}

type Config interface {
	kubectl.Config

	GetNamespaces() []string
}

func NewSyncer(cfg Config) Syncer {
	return &podSyncer{
		kubectl:    pkgkubectl.NewCLI(cfg, ""),
		namespaces: cfg.GetNamespaces(),
	}
}
