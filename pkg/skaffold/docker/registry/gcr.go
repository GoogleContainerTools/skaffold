/*
Copyright 2020 The Skaffold Authors

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

package registry

import (
	"fmt"
	"strings"
)

type GCRRegistry struct {
	domain  string
	project string
	paths   []string
}

func NewGCRRegistry(name string) (Registry, error) {
	p := strings.Split(name, "/")
	if len(p) < 2 {
		return nil, fmt.Errorf("not a valid GCR registry")
	}
	paths := []string{}
	if len(p) > 2 {
		paths = p[2:]
	}
	return &GCRRegistry{
		domain:  p[0],
		project: p[1],
		paths:   paths,
	}, nil
}

func (r *GCRRegistry) Prefix() string {
	return strings.Join(append([]string{"gcr.io", r.project}, r.paths...), "/")
}

func (r *GCRRegistry) Name() string {
	return strings.Join(append([]string{r.domain, r.project}, r.paths...), "/")
}

func (r *GCRRegistry) Update(reg Registry) Registry {
	switch t := reg.(type) {
	case *GCRRegistry:
		i := 0
		for ; i < len(t.paths)-1; i++ {
			if i > len(r.paths) || t.paths[i] != r.paths[i] {
				break
			}
			i++
		}
		return &GCRRegistry{
			domain:  t.domain,
			project: t.project,
			paths:   append(t.paths, r.paths[i:len(r.paths)]...),
		}
	default:
		return reg
	}
}

func (r *GCRRegistry) Type() string {
	return GCR
}
