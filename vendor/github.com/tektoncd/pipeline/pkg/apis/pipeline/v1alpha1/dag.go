/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import (
	"strings"

	"golang.org/x/xerrors"
)

// Node represents a Task in a pipeline.
type Node struct {
	// Task represent the PipelineTask in Pipeline
	Task PipelineTask
	// Prev represent all the Previous task Nodes for the current Task
	Prev []*Node
	// Next represent all the Next task Nodes for the current Task
	Next []*Node
}

// DAG represents the Pipeline DAG
type DAG struct {
	//Nodes represent map of PipelineTask name to Node in Pipeline DAG
	Nodes map[string]*Node
}

// Returns an empty Pipeline DAG
func newDAG() *DAG {
	return &DAG{Nodes: map[string]*Node{}}
}

func (g *DAG) addPipelineTask(t PipelineTask) (*Node, error) {
	if _, ok := g.Nodes[t.Name]; ok {
		return nil, xerrors.New("duplicate pipeline task")
	}
	newNode := &Node{
		Task: t,
	}
	g.Nodes[t.Name] = newNode
	return newNode, nil
}

func linkPipelineTasks(prev *Node, next *Node) error {
	// Check for self cycle
	if prev.Task.Name == next.Task.Name {
		return xerrors.Errorf("cycle detected; task %q depends on itself", next.Task.Name)
	}
	// Check if we are adding cycles.
	visited := map[string]bool{prev.Task.Name: true, next.Task.Name: true}
	path := []string{next.Task.Name, prev.Task.Name}
	if err := visit(next.Task.Name, prev.Prev, path, visited); err != nil {
		return xerrors.Errorf("cycle detected: %w", err)
	}
	next.Prev = append(next.Prev, prev)
	prev.Next = append(prev.Next, next)
	return nil
}

func visit(currentName string, nodes []*Node, path []string, visited map[string]bool) error {
	for _, n := range nodes {
		path = append(path, n.Task.Name)
		if _, ok := visited[n.Task.Name]; ok {
			return xerrors.New(getVisitedPath(path))
		}
		visited[currentName+"."+n.Task.Name] = true
		if err := visit(n.Task.Name, n.Prev, path, visited); err != nil {
			return err
		}
	}
	return nil
}

func getVisitedPath(path []string) string {
	// Reverse the path since we traversed the DAG using prev pointers.
	for i := len(path)/2 - 1; i >= 0; i-- {
		opp := len(path) - 1 - i
		path[i], path[opp] = path[opp], path[i]
	}
	return strings.Join(path, " -> ")
}

func addLink(pt PipelineTask, previousTask string, nodes map[string]*Node) error {
	prev, ok := nodes[previousTask]
	if !ok {
		return xerrors.Errorf("Task %s depends on %s but %s wasn't present in Pipeline", pt.Name, previousTask, previousTask)
	}
	next := nodes[pt.Name]
	if err := linkPipelineTasks(prev, next); err != nil {
		return xerrors.Errorf("Couldn't create link from %s to %s: %w", prev.Task.Name, next.Task.Name, err)
	}
	return nil
}

// BuildDAG returns a valid pipeline DAG. Returns error if the pipeline is invalid
func BuildDAG(tasks []PipelineTask) (*DAG, error) {
	d := newDAG()

	// Add all Tasks mentioned in the `PipelineSpec`
	for _, pt := range tasks {
		if _, err := d.addPipelineTask(pt); err != nil {
			return nil, xerrors.Errorf("task %s is already present in DAG, can't add it again: %w", pt.Name, err)
		}
	}
	// Process all from and runAfter constraints to add task dependency
	for _, pt := range tasks {
		for _, previousTask := range pt.RunAfter {
			if err := addLink(pt, previousTask, d.Nodes); err != nil {
				return nil, xerrors.Errorf("couldn't add link between %s and %s: %w", pt.Name, previousTask, err)
			}
		}
		if pt.Resources != nil {
			for _, rd := range pt.Resources.Inputs {
				for _, previousTask := range rd.From {
					if err := addLink(pt, previousTask, d.Nodes); err != nil {
						return nil, xerrors.Errorf("couldn't add link between %s and %s: %w", pt.Name, previousTask, err)
					}
				}
			}
		}
	}
	return d, nil
}
