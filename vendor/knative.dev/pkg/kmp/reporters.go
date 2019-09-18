/*
Copyright 2019 The Knative Authors

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

package kmp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// FieldListReporter implements the cmp.Reporter interface. It keeps
// track of the field names that differ between two structs and reports
// them through the Fields() function.
type FieldListReporter struct {
	path       cmp.Path
	fieldNames []string
}

// PushStep implements the cmp.Reporter.
func (r *FieldListReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

// fieldName returns a readable name for the field. If the field has JSON annotations it
// returns the JSON key. If the field does not have JSON annotations or the JSON annotation
// marks the field as ignored it returns the field's go name
func (r *FieldListReporter) fieldName() string {
	if len(r.path) < 2 {
		return r.path.Index(0).String()
	} else {
		fieldName := strings.TrimPrefix(r.path.Index(1).String(), ".")
		// Prefer JSON name to fieldName if it exists
		structField, exists := r.path.Index(0).Type().FieldByName(fieldName)
		if exists {
			tag := structField.Tag.Get("json")
			if tag != "" && tag != "-" {
				return strings.SplitN(tag, ",", 2)[0]
			}

		}
		return fieldName
	}
}

// Report implements the cmp.Reporter.
func (r *FieldListReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}
	name := r.fieldName()
	// Only append elements we don't already have.
	for _, v := range r.fieldNames {
		if name == v {
			return
		}
	}
	r.fieldNames = append(r.fieldNames, name)
}

// PopStep implements cmp.Reporter.
func (r *FieldListReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

// Fields returns the field names that differed between the two
// objects after calling cmp.Equal with the FieldListReporter. Field names
// are returned in alphabetical order.
func (r *FieldListReporter) Fields() []string {
	sort.Strings(r.fieldNames)
	return r.fieldNames
}

// ShortDiffReporter implements the cmp.Reporter interface. It reports
// on fields which have diffing values in a short zero-context, unified diff
// format.
type ShortDiffReporter struct {
	path  cmp.Path
	diffs []string
	err   error
}

// PushStep implements the cmp.Reporter.
func (r *ShortDiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

// Report implements the cmp.Reporter.
func (r *ShortDiffReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}
	cur := r.path.Last()
	vx, vy := cur.Values()
	t := cur.Type()
	var diff string
	// Prefix struct values with the types to add clarity in output
	if !vx.IsValid() && !vy.IsValid() {
		r.err = fmt.Errorf("unable to diff %+v and %+v on path %#v", vx, vy, r.path)
	} else {
		diff = fmt.Sprintf("%#v:\n", r.path)
		if vx.IsValid() {
			diff += r.diffString("-", t, vx)
		}
		if vy.IsValid() {
			diff += r.diffString("+", t, vy)
		}
	}
	r.diffs = append(r.diffs, diff)
}

func (r *ShortDiffReporter) diffString(diffType string, t reflect.Type, v reflect.Value) string {
	if t.Kind() == reflect.Struct {
		return fmt.Sprintf("\t%s: %+v: \"%+v\"\n", diffType, t, v)
	} else {
		return fmt.Sprintf("\t%s: \"%+v\"\n", diffType, v)
	}
}

// PopStep implements the cmp.Reporter.
func (r *ShortDiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

// Diff returns the generated short diff for this object.
// cmp.Equal should be called before this method.
func (r *ShortDiffReporter) Diff() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return strings.Join(r.diffs, ""), nil
}
