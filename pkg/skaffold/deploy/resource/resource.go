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

package resource

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	tabHeader = " -"
)

type Status struct {
	err       error
	details   string
	updated   bool
	completed bool
}

func (rs *Status) Error() error {
	return rs.err
}

func (rs *Status) Equals(other *Status) bool {
	if util.Trim(rs.details) != util.Trim(other.details) {
		return false
	}
	if rs.err == other.err {
		return true
	}
	return rs.err.Error() == other.err.Error()
}

func (rs *Status) String() string {
	if rs.err != nil {
		return util.Trim(rs.err.Error())
	}
	return rs.details
}

func NewStatus(msg string, err error) Status {
	return Status{
		details: msg,
		err:     err,
	}
}
