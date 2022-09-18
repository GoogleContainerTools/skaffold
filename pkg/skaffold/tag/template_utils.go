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

package tag

import (
	"reflect"
	"text/template"
	"text/template/parse"
)

// GetTemplateFields returns a list of template variables.
func GetTemplateFields(t *template.Template) []string {
	fields := getNodeFields(t.Root.Nodes...)
	u := make([]string, 0, len(fields))
	m := make(map[string]bool)

	for _, f := range fields {
		if _, ok := m[f]; !ok {
			m[f] = true
			u = append(u, f)
		}
	}
	return u
}

// getNodeFields returns a list of template node list fields.
func getNodeFields(nodes ...parse.Node) []string {
	var fields []string
	for _, node := range nodes {
		fields = append(fields, getFields(node)...)
	}
	return fields
}

// getFields returns a list of template node fields.
func getFields(node parse.Node) []string {
	var fields []string
	if reflect.ValueOf(node).IsNil() {
		return fields
	}
	switch n := node.(type) {
	case *parse.FieldNode:
		return n.Ident
	case *parse.ActionNode:
		return getFields(n.Pipe)
	case *parse.TemplateNode:
		return getFields(n.Pipe)
	case *parse.CommandNode:
		return getNodeFields(n.Args...)
	case *parse.ListNode:
		return getNodeFields(n.Nodes...)
	case *parse.IfNode:
		return getNodeFields(n.Pipe, n.List, n.ElseList)
	case *parse.RangeNode:
		return getNodeFields(n.Pipe, n.List, n.ElseList)
	case *parse.WithNode:
		return getNodeFields(n.Pipe, n.List, n.ElseList)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			fields = append(fields, getFields(cmd)...)
		}
	}
	return fields
}
