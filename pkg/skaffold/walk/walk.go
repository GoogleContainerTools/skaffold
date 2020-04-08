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

package walk

import (
	"os"

	"github.com/karrick/godirwalk"
)

// Dirent stores the name and type of a file system entry.
type Dirent interface {
	IsDir() bool
	Name() string
}

// Predicate represents a predicate on file system entries.
// Given a file's path and information, it returns `true`
// when the predicate is matched. It can also return a `filepath.SkipDir`
// error to skip a directory and its children altogether.
type Predicate func(path string, info Dirent) (bool, error)

// Action is a function that takes a file's path and information,
// and optionally returns an error.
type Action func(path string, info Dirent) error

type Builder interface {
	// Options
	Unsorted() Builder

	// Predicates
	When(Predicate) Builder
	WhenIsDir() Builder
	WhenIsFile() Builder
	WhenHasName(string) Builder

	// Actions
	Do(Action) error
	MustDo(Action)
	AppendPaths(*[]string) error
	CollectPaths() ([]string, error)
}

type builder struct {
	dir       string
	unsorted  bool
	predicate Predicate
}

func From(dir string) Builder {
	return &builder{
		dir:       dir,
		unsorted:  false,
		predicate: func(string, Dirent) (bool, error) { return true, nil },
	}
}

func (w *builder) Unsorted() Builder {
	w.unsorted = true
	return w
}

func (w *builder) When(predicate Predicate) Builder {
	w.predicate = and(w.predicate, predicate)
	return w
}

func (w *builder) WhenIsFile() Builder {
	return w.When(isFile)
}

func (w *builder) WhenIsDir() Builder {
	return w.When(isDir)
}

func (w *builder) WhenHasName(name string) Builder {
	return w.When(hasName(name))
}

func (w *builder) AppendPaths(values *[]string) error {
	return w.Do(appendPaths(values))
}

func (w *builder) CollectPaths() ([]string, error) {
	var paths []string
	err := w.Do(appendPaths(&paths))
	return paths, err
}

func (w *builder) Do(action Action) error {
	info, err := os.Lstat(w.dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		match, err := w.predicate(w.dir, info)
		if !match || err != nil {
			return err
		}

		return action(w.dir, info)
	}

	return godirwalk.Walk(w.dir, &godirwalk.Options{
		Unsorted: w.unsorted,
		Callback: func(path string, info *godirwalk.Dirent) error {
			match, err := w.predicate(path, info)
			if !match || err != nil {
				return err
			}

			return action(path, info)
		},
	})
}

func (w *builder) MustDo(action Action) {
	if err := w.Do(action); err != nil {
		panic("unable to list files: " + err.Error())
	}
}

// Predicates

func hasName(name string) Predicate {
	return func(_ string, info Dirent) (bool, error) {
		return info.Name() == name, nil
	}
}

func isDir(_ string, info Dirent) (bool, error) {
	return info.IsDir(), nil
}

func isFile(_ string, info Dirent) (bool, error) {
	return !info.IsDir(), nil
}

func and(p1, p2 Predicate) Predicate {
	return func(path string, info Dirent) (bool, error) {
		match, err := p1(path, info)
		if !match || err != nil {
			return false, err
		}

		return p2(path, info)
	}
}

// Actions

func appendPaths(values *[]string) Action {
	return func(path string, _ Dirent) error {
		*values = append(*values, path)
		return nil
	}
}
