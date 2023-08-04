/*
 *
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package common implements common functionality for dealing with text/template.
package common

import (
	"io"
	"reflect"
	"sync"
	"text/template"
	"text/template/parse"
	"unicode"
	"unicode/utf8"
)

const textTemplateRemediationFuncName = "ApplyInjectionDetection"

// FuncMap to register new template objects with.
var FuncMap = map[string]any{
	textTemplateRemediationFuncName: textTemplateRemediationFunc,
	"StructuralData":                echo,
}

func echo(in interface{}) interface{} {
	return in
}

// EchoString is a nop string callback
func EchoString(in string) string {
	return in
}

// BaselineString is a string callback function that just returns a constant string,
// used to get a baseline of how the resultant YAML is structured.
func BaselineString(string) string {
	return "baseline"
}

// stringCallback provides the callback for how strings should be manipulated before
// being pasted into the template execution result.
var stringCallback func(string) string
var stringCallbackLock sync.Mutex

func textTemplateRemediationFunc(data interface{}) interface{} {
	return DeepCopyMutateStrings(data, stringCallback)
}

// ExecuteWithCallback performs an execution on a callback-applied template
// (WalkApplyFuncToNonDeclaractiveActions) with a specified callback.
func ExecuteWithCallback(tmpl *template.Template, cb func(string) string, result io.Writer, data interface{}) error {
	stringCallbackLock.Lock()
	defer stringCallbackLock.Unlock()
	stringCallback = cb

	return tmpl.Execute(result, data)
}

func makePointer(data interface{}) interface{} {
	rtype := reflect.New(reflect.TypeOf(data))
	rtype.Elem().Set(reflect.ValueOf(data))
	return rtype.Interface()
}

func dereference(data interface{}) interface{} {
	return reflect.ValueOf(data).Elem().Interface()
}

// DeepCopyMutateStrings performs a deep copy, but mutates any strings according to the mutation callback.
func DeepCopyMutateStrings(data interface{}, mutateF func(string) string) interface{} {
	var r interface{}

	if data == nil {
		return nil
	}

	switch reflect.TypeOf(data).Kind() {
	case reflect.Ptr:
		p := reflect.ValueOf(data)
		if p.IsNil() {
			r = data
		} else {
			c := DeepCopyMutateStrings(dereference(data), mutateF)
			r = makePointer(c)

			// Sometimes we accidentally introduce one too minterface{} layers of indirection (seems related to protobuf generated fields like ReleaseNamespace *ReleaseNamespace `... reflect:"unexport"`)
			if reflect.TypeOf(r) != reflect.TypeOf(data) {
				r = c
			}
		}
	case reflect.String:
		return mutateF(reflect.ValueOf(data).String())
	case reflect.Slice, reflect.Array:
		rc := reflect.MakeSlice(reflect.TypeOf(data), reflect.ValueOf(data).Len(), reflect.ValueOf(data).Len())
		for i := 0; i < reflect.ValueOf(data).Len(); i++ {
			rc.Index(i).Set(reflect.ValueOf(DeepCopyMutateStrings(reflect.ValueOf(data).Index(i).Interface(), mutateF)))
		}
		r = rc.Interface()
	case reflect.Map:
		rc := reflect.MakeMap(reflect.TypeOf(data))
		dataIter := reflect.ValueOf(data).MapRange()
		for dataIter.Next() {
			rc.SetMapIndex(dataIter.Key(), reflect.ValueOf(DeepCopyMutateStrings(dataIter.Value().Interface(), mutateF)))
		}
		r = rc.Interface()
	case reflect.Struct:
		s := reflect.New(reflect.TypeOf(data))

		t := reflect.TypeOf(data)
		v := reflect.ValueOf(data)
		n := v.NumField()
		for i := 0; i < n; i++ {
			r, _ := utf8.DecodeRuneInString(t.Field(i).Name)

			// Don't copy unexported fields
			if unicode.IsUpper(r) {
				reflect.Indirect(s).Field(i).Set(
					reflect.ValueOf(DeepCopyMutateStrings(v.Field(i).Interface(), mutateF)),
				)
			}
		}

		r = s.Interface()
	default:
		// No other types need special handling (int, bool, etc)
		r = data
	}

	return r
}

func applyPipeCmds(cmds []*parse.CommandNode) {
	for _, c := range cmds {
		newArgs := make([]parse.Node, 0)
		for i, a := range c.Args {
			switch a := a.(type) {
			case *parse.DotNode, *parse.FieldNode, *parse.VariableNode:
				if i == 0 && len(c.Args) > 1 {
					// If this is the first "argument" of multiple, then it is really a function
					newArgs = append(newArgs, a)
				} else {
					// If this node is an argument to a call to "StructuralData", then pass it through as-is
					switch identifier := c.Args[0].(type) {
					case *parse.IdentifierNode:
						if identifier.Ident == "StructuralData" {
							newArgs = append(newArgs, a)
							continue
						}
					}

					newPipe := &parse.PipeNode{NodeType: parse.NodePipe, Decl: nil}
					newPipe.Cmds = []*parse.CommandNode{
						&parse.CommandNode{NodeType: parse.NodeCommand, Args: []parse.Node{a}},
						&parse.CommandNode{NodeType: parse.NodeCommand, Args: []parse.Node{
							&parse.IdentifierNode{
								NodeType: parse.NodeIdentifier,
								Ident:    textTemplateRemediationFuncName,
							},
						}},
					}
					newArgs = append(newArgs, newPipe)
				}
			case *parse.PipeNode:
				applyPipeCmds(a.Cmds)
				newArgs = append(newArgs, a)
			default:
				newArgs = append(newArgs, a)
			}
		}

		c.Args = newArgs
	}
}

func branchNode(node parse.Node) *parse.BranchNode {
	switch node := node.(type) {
	case *parse.IfNode:
		return &node.BranchNode
	case *parse.RangeNode:
		return &node.BranchNode
	case *parse.WithNode:
		return &node.BranchNode
	}

	return nil
}

// WalkApplyFuncToNonDeclaractiveActions walks the AST, applying a pipeline function to interface{} "paste" nodes (non-declarative action nodes)
func WalkApplyFuncToNonDeclaractiveActions(template *template.Template, node parse.Node) {
	switch node := node.(type) {
	case *parse.ActionNode:
		// Non-declarative actions are paste actions
		if len(node.Pipe.Decl) == 0 {
			applyPipeCmds(node.Pipe.Cmds)
		}

	case *parse.IfNode, *parse.RangeNode, *parse.WithNode:
		nodeBranch := branchNode(node)
		WalkApplyFuncToNonDeclaractiveActions(template, nodeBranch.List)
		if nodeBranch.ElseList != nil {
			WalkApplyFuncToNonDeclaractiveActions(template, nodeBranch.ElseList)
		}
	case *parse.ListNode:
		for _, node := range node.Nodes {
			WalkApplyFuncToNonDeclaractiveActions(template, node)
		}
	case *parse.TemplateNode:
		tmpl := template.Lookup(node.Name)
		if tmpl != nil {
			treeCopy := tmpl.Tree.Copy()
			WalkApplyFuncToNonDeclaractiveActions(tmpl, treeCopy.Root)
			template.AddParseTree(node.Name, treeCopy)
		}
	}
}
