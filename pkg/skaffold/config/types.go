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

import (
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// StringOrUndefined holds the value of a flag of type `string`,
// that's by default `undefined`.
// We use this instead of just `string` to differentiate `undefined`
// and `empty string` values.
type StringOrUndefined struct {
	value *string
}

func (s StringOrUndefined) Equal(o StringOrUndefined) bool {
	if s.value == nil || o.value == nil {
		return s.value == o.value
	}

	return *s.value == *o.value
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

func (s *StringOrUndefined) SetNil() error {
	s.value = nil
	return nil
}

func (s *StringOrUndefined) String() string {
	if s.value == nil {
		return ""
	}
	return *s.value
}

func NewStringOrUndefined(v *string) StringOrUndefined {
	return StringOrUndefined{value: v}
}

// BoolOrUndefined holds the value of a flag of type `bool`,
// that's by default `undefined`.
// We use this instead of just `bool` to differentiate `undefined`
// and `false` values.
type BoolOrUndefined struct {
	value *bool
}

func (s *BoolOrUndefined) Type() string {
	return "bool"
}

func (s *BoolOrUndefined) Value() *bool {
	return s.value
}

func (s *BoolOrUndefined) Set(v string) error {
	switch v {
	case "true":
		s.value = util.BoolPtr(true)
	case "false":
		s.value = util.BoolPtr(false)
	default:
		s.value = nil
	}
	return nil
}

func (s *BoolOrUndefined) SetNil() error {
	s.value = nil
	return nil
}

func (s *BoolOrUndefined) String() string {
	b := s.value
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}

func NewBoolOrUndefined(v *bool) BoolOrUndefined {
	return BoolOrUndefined{value: v}
}

// Muted lists phases for which logs are muted.
type Muted struct {
	Phases []string
}

// IntOrUndefined holds the value of a flag of type `int`,
// that's by default `undefined`.
// We use this instead of just `int` to differentiate `undefined`
// and `zero` values.
type IntOrUndefined struct {
	value *int
}

func (s *IntOrUndefined) Type() string {
	return "int"
}

func (s *IntOrUndefined) Value() *int {
	return s.value
}

func (s *IntOrUndefined) Set(v string) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	s.value = &i
	return nil
}

func (s *IntOrUndefined) SetNil() error {
	s.value = nil
	return nil
}

func (s *IntOrUndefined) String() string {
	if s.value == nil {
		return ""
	}
	return strconv.Itoa(*s.value)
}

func NewIntOrUndefined(v *int) IntOrUndefined {
	return IntOrUndefined{value: v}
}

func (m Muted) MuteBuild() bool       { return m.mute("build") }
func (m Muted) MuteRender() bool      { return m.mute("render") }
func (m Muted) MuteTest() bool        { return m.mute("test") }
func (m Muted) MuteStatusCheck() bool { return m.mute("status-check") }
func (m Muted) MuteDeploy() bool      { return m.mute("deploy") }
func (m Muted) mute(phase string) bool {
	return stringslice.Contains(m.Phases, phase) || stringslice.Contains(m.Phases, "all")
}

type Cluster struct {
	Local       bool
	PushImages  bool
	LoadImages  bool
	DefaultRepo StringOrUndefined
}
