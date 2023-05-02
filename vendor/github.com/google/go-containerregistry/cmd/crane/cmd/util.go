// Copyright 2020 Google LLC All Rights Reserved.
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

package cmd

import (
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type platformsValue struct {
	platforms []v1.Platform
}

func (ps *platformsValue) Set(platform string) error {
	if ps.platforms == nil {
		ps.platforms = []v1.Platform{}
	}
	p, err := parsePlatform(platform)
	if err != nil {
		return err
	}
	pv := platformValue{p}
	ps.platforms = append(ps.platforms, *pv.platform)
	return nil
}

func (ps *platformsValue) String() string {
	ss := make([]string, 0, len(ps.platforms))
	for _, p := range ps.platforms {
		ss = append(ss, p.String())
	}
	return strings.Join(ss, ",")
}

func (ps *platformsValue) Type() string {
	return "platform(s)"
}

type platformValue struct {
	platform *v1.Platform
}

func (pv *platformValue) Set(platform string) error {
	p, err := parsePlatform(platform)
	if err != nil {
		return err
	}
	pv.platform = p
	return nil
}

func (pv *platformValue) String() string {
	return platformToString(pv.platform)
}

func (pv *platformValue) Type() string {
	return "platform"
}

func platformToString(p *v1.Platform) string {
	if p == nil {
		return "all"
	}
	return p.String()
}

func parsePlatform(platform string) (*v1.Platform, error) {
	if platform == "all" {
		return nil, nil
	}

	return v1.ParsePlatform(platform)
}
