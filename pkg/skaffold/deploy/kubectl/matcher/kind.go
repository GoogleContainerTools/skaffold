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

package matcher

import (
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	kindKey = "kind"
)

// Kind matches the value of "Kind" key with values not listed in the forbiddenValues.
type Kind struct {
	forbiddenValues map[string]struct{}
}

func (k *Kind) Matches(v interface{}) bool {
	if str, ok := k.getValue(v); ok {
		return k.notInBlacklist(str)
	}
	logrus.Debugf("%v is type %T but type string expected for key `%s`. skipping value match.", v, v, kindKey)
	return true
}

func (k *Kind) getValue(v interface{}) (string, bool) {
	value, ok := v.(string)
	return value, ok
}

func (k *Kind) notInBlacklist(s string) bool {
	_, ok := k.forbiddenValues[s]
	return !ok
}

func (k *Kind) IsMatchKey(key string) bool {
	return strings.ToLower(key) == kindKey
}

func New(values []string) *Kind {
	m := map[string]struct{}{}
	for _, v := range values {
		m[v] = struct{}{}
	}
	return &Kind{
		forbiddenValues: m,
	}
}
