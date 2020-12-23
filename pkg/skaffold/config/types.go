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

package config

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

// StringOrUndefined holds the value of a flag of type `string`,
// that's by default `undefined`.
// We use this instead of just `string` to differentiate `undefined`
// and `empty string` values.
type StringOrUndefined struct {
	value *string
}

func (s *StringOrUndefined) Type() string {
	return "string"
}

func (s *StringOrUndefined) Value() *string {
	return s.value
}

func (s *StringOrUndefined) Set(v string) error {
	s.value = &v
	return nil
}

func (s *StringOrUndefined) String() string {
	if s.value == nil {
		return ""
	}
	return *s.value
}

// Muted lists phases for which logs are muted.
type Muted struct {
	Phases []string
}

func (m Muted) MuteBuild() bool       { return m.mute("build") }
func (m Muted) MuteTest() bool        { return m.mute("test") }
func (m Muted) MuteStatusCheck() bool { return m.mute("status-check") }
func (m Muted) MuteDeploy() bool      { return m.mute("deploy") }
func (m Muted) mute(phase string) bool {
	return util.StrSliceContains(m.Phases, phase) || util.StrSliceContains(m.Phases, "all")
}

type Cluster struct {
	Local      bool
	PushImages bool
	LoadImages bool
}
