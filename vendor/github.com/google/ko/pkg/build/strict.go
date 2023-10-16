// Copyright 2020 ko Build Authors All Rights Reserved.
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

package build

import "strings"

// StrictScheme is a prefix that can be placed on import paths that users
// think MUST be supported references.
const StrictScheme = "ko://"

type reference struct {
	strict bool
	path   string
}

func newRef(s string) reference {
	return reference{
		strict: strings.HasPrefix(s, StrictScheme),
		path:   strings.TrimPrefix(s, StrictScheme),
	}
}

func (r reference) IsStrict() bool {
	return r.strict
}

func (r reference) Path() string {
	return r.path
}

func (r reference) String() string {
	if r.IsStrict() {
		return StrictScheme + r.Path()
	}
	return r.Path()
}
