// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Function defines a function which mutates or validates a collection of configuration
// To create a structured validation result, return a Result as the error.
type Function func() error

// Result defines a function result which will be set on the emitted ResourceList
type Result struct {
	// Name is the name of the function creating the result
	Name string `yaml:"name,omitempty"`

	// Items are the individual results
	Items []Item `yaml:"items,omitempty"`
}

// Severity indicates the severity of the result
type Severity string

const (
	// Error indicates the result is an error.  Will cause the function to exit non-0.
	Error Severity = "error"
	// Warning indicates the result is a warning
	Warning Severity = "warning"
	// Info indicates the result is an informative message
	Info Severity = "info"
)

// Item defines a validation result
type Item struct {
	// Message is a human readable message
	Message string `yaml:"message,omitempty"`

	// Severity is the severity of the
	Severity Severity `yaml:"severity,omitempty"`

	// ResourceRef is a reference to a resource
	ResourceRef yaml.ResourceMeta `yaml:"resourceRef,omitempty"`

	Field Field `yaml:"field,omitempty"`

	File File `yaml:"file,omitempty"`
}

// File references a file containing a resource
type File struct {
	// Path is relative path to the file containing the resource
	Path string `yaml:"path,omitempty"`

	// Index is the index into the file containing the resource
	// (i.e. if there are multiple resources in a single file)
	Index int `yaml:"index,omitempty"`
}

// Field references a field in a resource
type Field struct {
	// Path is the field path
	Path string `yaml:"path,omitempty"`

	// CurrentValue is the current field value
	CurrentValue string `yaml:"currentValue,omitempty"`

	// SuggestedValue is the suggested field value
	SuggestedValue string `yaml:"suggestedValue,omitempty"`
}

// Error implement error
func (e Result) Error() string {
	var msgs []string
	for _, i := range e.Items {
		msgs = append(msgs, i.Message)
	}
	return strings.Join(msgs, "\n\n")
}

// ExitCode provides the exit code based on the result
func (e Result) ExitCode() int {
	for _, i := range e.Items {
		if i.Severity == Error {
			return 1
		}
	}
	return 0
}
