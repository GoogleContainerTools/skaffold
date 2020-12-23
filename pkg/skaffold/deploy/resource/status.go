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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/proto"
)

type Status struct {
	ae       proto.ActionableErr
	changed  bool
	reported bool
}

func (rs Status) Error() error {
	if rs.ae.ErrCode == proto.StatusCode_STATUSCHECK_SUCCESS {
		return nil
	}
	return fmt.Errorf(rs.ae.Message)
}

func (rs Status) ActionableError() proto.ActionableErr {
	return rs.ae
}

func (rs Status) String() string {
	return rs.ae.Message
}

func (rs Status) Equal(other Status) bool {
	return rs.ae.Message == other.ae.Message && rs.ae.ErrCode == other.ae.ErrCode
}

func newStatus(ae proto.ActionableErr) Status {
	return Status{
		ae:      ae,
		changed: true,
	}
}
