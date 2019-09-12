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
)

type Base struct {
	name      string
	namespace string
	rType     string
	status    Status
	done      bool
}


func (b *Base) String() string {
	return fmt.Sprintf("%s:%s/%s", b.namespace, b.rType, b.name)
}

func (b *Base) Name() string {
	return b.name
}

func (b *Base) Status() Status {
	return b.status
}

func (b *Base) UpdateStatus(details string, err error) {
	updated := newStatus(details, err)
	if !b.status.Equal(updated) {
		b.status = updated
	}
}

func (b *Base) IsDone() bool {
	return b.done
}

func (b *Base) MarkDone() {
	b.done = true
}

func (b *Base) ReportSinceLastUpdated() string {
	if b.status.reported {
		return ""
	}
	b.status.reported = true
	return fmt.Sprintf("%s %s", b, b.status)
}


// For testing
func (b *Base) WithStatus(details string, err error) *Base {
	b.UpdateStatus(details, err)
	return b
}

func (b *Base) WithDone(details string, err error) *Base {
	b.UpdateStatus(details, err)
	b.done = true
	return b
}