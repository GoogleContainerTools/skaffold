// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceMatcher is implemented by types designed for use in or as selectors.
type ResourceMatcher interface {
	// kio.Filter applies the matcher to multiple resources.
	// This makes individual matchers usable as selectors directly.
	kio.Filter
	// Match returns true if the given resource matches the matcher's configuration.
	Match(node *yaml.RNode) bool
}

// ResourceMatcherFunc converts a compliant function into a ResourceMatcher
type ResourceMatcherFunc func(node *yaml.RNode) bool

// Match runs the ResourceMatcherFunc on the given node.
func (m ResourceMatcherFunc) Match(node *yaml.RNode) bool {
	return m(node)
}

// Filter applies ResourceMatcherFunc to a list of items, returning only those that match.
func (m ResourceMatcherFunc) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	// MatchAll or MatchAny doesn't really matter here since there is only one matcher (m).
	return MatchAll(m).Filter(items)
}

// ResourceTemplateMatcher is implemented by ResourceMatcher types that accept text templates as
// part of their configuration.
type ResourceTemplateMatcher interface {
	// ResourceMatcher makes matchers usable in or as selectors.
	ResourceMatcher
	// DefaultTemplateData is used to pass default template values down a chain of matchers.
	DefaultTemplateData(interface{})
	// InitTemplates is used to render the templates in selectors that support
	// ResourceTemplateMatcher. The selector should call this exactly once per filter
	// operation, before beginning match comparisons.
	InitTemplates() error
}

// ContainerNameMatcher returns a function that returns true if the "name" field
// of the provided container node matches one of the given container names.
// If no names are provided, the function always returns true.
// Note that this is not a ResourceMatcher, since the node it matches against must be
// container-level (e.g. "name", "env" and "image" would be top level fields).
func ContainerNameMatcher(names ...string) func(node *yaml.RNode) bool {
	namesSet := sets.String{}
	namesSet.Insert(names...)
	return func(node *yaml.RNode) bool {
		if len(namesSet) == 0 {
			return true
		}
		f := node.Field("name")
		if f == nil {
			return false
		}
		return namesSet.Has(yaml.GetValue(f.Value))
	}
}

// NameMatcher matches resources whose metadata.name is equal to one of the provided values.
// e.g. `NameMatcher("foo", "bar")` matches if `metadata.name` is either "foo" or "bar".
//
// NameMatcher supports templating.
// e.g. `NameMatcher("{{.AppName}}")` will match `metadata.name` "foo" if TemplateData is
// `struct{ AppName string }{ AppName: "foo" }`
func NameMatcher(names ...string) ResourceTemplateMatcher {
	return &TemplatedMetaSliceMatcher{
		Templates: names,
		MetaMatcher: func(names sets.String, meta yaml.ResourceMeta) bool {
			return names.Has(meta.Name)
		},
	}
}

// NamespaceMatcher matches resources whose metadata.namespace is equal to one of the provided values.
// e.g. `NamespaceMatcher("foo", "bar")` matches if `metadata.namespace` is either "foo" or "bar".
//
// NamespaceMatcher supports templating.
// e.g. `NamespaceMatcher("{{.AppName}}")` will match `metadata.namespace` "foo" if TemplateData is
// `struct{ AppName string }{ AppName: "foo" }`
func NamespaceMatcher(names ...string) ResourceTemplateMatcher {
	return &TemplatedMetaSliceMatcher{
		Templates: names,
		MetaMatcher: func(names sets.String, meta yaml.ResourceMeta) bool {
			return names.Has(meta.Namespace)
		},
	}
}

// KindMatcher matches resources whose kind is equal to one of the provided values.
// e.g. `KindMatcher("foo", "bar")` matches if `kind` is either "foo" or "bar".
//
// KindMatcher supports templating.
// e.g. `KindMatcher("{{.TargetKind}}")` will match `kind` "foo" if TemplateData is
// `struct{ TargetKind string }{ TargetKind: "foo" }`
func KindMatcher(names ...string) ResourceTemplateMatcher {
	return &TemplatedMetaSliceMatcher{
		Templates: names,
		MetaMatcher: func(names sets.String, meta yaml.ResourceMeta) bool {
			return names.Has(meta.Kind)
		},
	}
}

// APIVersionMatcher matches resources whose kind is equal to one of the provided values.
// e.g. `APIVersionMatcher("foo/v1", "bar/v1")` matches if `apiVersion` is either "foo/v1" or
// "bar/v1".
//
// APIVersionMatcher supports templating.
// e.g. `APIVersionMatcher("{{.TargetAPI}}")` will match `apiVersion` "foo/v1" if TemplateData is
// `struct{ TargetAPI string }{ TargetAPI: "foo/v1" }`
func APIVersionMatcher(names ...string) ResourceTemplateMatcher {
	return &TemplatedMetaSliceMatcher{
		Templates: names,
		MetaMatcher: func(names sets.String, meta yaml.ResourceMeta) bool {
			return names.Has(meta.APIVersion)
		},
	}
}

// GVKMatcher matches resources whose API group, version and kind match one of the provided values.
// e.g. `GVKMatcher("foo/v1/Widget", "bar/v1/App")` matches if `apiVersion` concatenated with `kind`
// is either "foo/v1/Widget" or "bar/v1/App".
//
// GVKMatcher supports templating.
// e.g. `GVKMatcher("{{.TargetAPI}}")` will match "foo/v1/Widget" if TemplateData is
// `struct{ TargetAPI string }{ TargetAPI: "foo/v1/Widget" }`
func GVKMatcher(names ...string) ResourceTemplateMatcher {
	return &TemplatedMetaSliceMatcher{
		Templates: names,
		MetaMatcher: func(names sets.String, meta yaml.ResourceMeta) bool {
			gvk := strings.Join([]string{meta.APIVersion, meta.Kind}, "/")
			return names.Has(gvk)
		},
	}
}

// TemplatedMetaSliceMatcher is a utility type for constructing matchers that compare resource
// metadata to a slice of (possibly templated) strings.
type TemplatedMetaSliceMatcher struct {
	// Templates is the list of possibly templated strings to compare to.
	Templates []string
	// values is the set of final (possibly rendered) strings to compare to.
	values sets.String
	// TemplateData is the data to use in template rendering.
	// Rendering will not take place if it is nil when InitTemplates is called.
	TemplateData interface{}
	// MetaMatcher is a function that returns true if the given resource metadata matches at
	// least one of the given names.
	// The matcher implemented using TemplatedMetaSliceMatcher can compare names to any meta field.
	MetaMatcher func(names sets.String, meta yaml.ResourceMeta) bool
}

// Match parses the resource node's metadata and delegates matching logic to the provided
// MetaMatcher func. This allows ResourceMatchers build with TemplatedMetaSliceMatcher to match
// against any field in resource metadata.
func (m *TemplatedMetaSliceMatcher) Match(node *yaml.RNode) bool {
	var err error
	meta, err := node.GetMeta()
	if err != nil {
		return false
	}
	return m.MetaMatcher(m.values, meta)
}

// Filter applies the matcher to a list of items, returning only those that match.
func (m *TemplatedMetaSliceMatcher) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	// AndSelector or OrSelector doesn't really matter here since there is only one matcher (m).
	s := AndSelector{Matchers: []ResourceMatcher{m}, TemplateData: m.TemplateData}
	return s.Filter(items)
}

// DefaultTemplateData sets TemplateData to the provided default values if it has not already
// been set.
func (m *TemplatedMetaSliceMatcher) DefaultTemplateData(data interface{}) {
	if m.TemplateData == nil {
		m.TemplateData = data
	}
}

// InitTemplates is used to render any templates the selector's list of strings may contain
// before the selector is applied. It should be called exactly once per filter
// operation, before beginning match comparisons.
func (m *TemplatedMetaSliceMatcher) InitTemplates() error {
	values, err := templatizeSlice(m.Templates, m.TemplateData)
	if err != nil {
		return errors.Wrap(err)
	}
	m.values = sets.String{}
	m.values.Insert(values...)
	return nil
}

var _ ResourceTemplateMatcher = &TemplatedMetaSliceMatcher{}

// LabelMatcher matches resources that are labelled with all of the provided key-value pairs.
// e.g. `LabelMatcher(map[string]string{"app": "foo", "env": "prod"})` matches resources labelled
// app=foo AND env=prod.
//
// LabelMatcher supports templating.
// e.g. `LabelMatcher(map[string]string{"app": "{{ .AppName}}"})` will match label app=foo if
// TemplateData is `struct{ AppName string }{ AppName: "foo" }`
func LabelMatcher(labels map[string]string) ResourceTemplateMatcher {
	return &TemplatedMetaMapMatcher{
		Templates: labels,
		MetaMatcher: func(labels map[string]string, meta yaml.ResourceMeta) bool {
			return compareMaps(labels, meta.Labels)
		},
	}
}

func compareMaps(desired, actual map[string]string) bool {
	for k := range desired {
		// actual either doesn't have the key or has the wrong value for it
		if actual[k] != desired[k] {
			return false
		}
	}
	return true
}

// AnnotationMatcher matches resources that are annotated with all of the provided key-value pairs.
// e.g. `AnnotationMatcher(map[string]string{"app": "foo", "env": "prod"})` matches resources
// annotated app=foo AND env=prod.
//
// AnnotationMatcher supports templating.
// e.g. `AnnotationMatcher(map[string]string{"app": "{{ .AppName}}"})` will match label app=foo if
// TemplateData is `struct{ AppName string }{ AppName: "foo" }`
func AnnotationMatcher(ann map[string]string) ResourceTemplateMatcher {
	return &TemplatedMetaMapMatcher{
		Templates: ann,
		MetaMatcher: func(ann map[string]string, meta yaml.ResourceMeta) bool {
			return compareMaps(ann, meta.Annotations)
		},
	}
}

// TemplatedMetaMapMatcher is a utility type for constructing matchers that compare resource
// metadata to a map of (possibly templated) key-value pairs.
type TemplatedMetaMapMatcher struct {
	// Templates is the list of possibly templated strings to compare to.
	Templates map[string]string
	// values is the map of final (possibly rendered) strings to compare to.
	values map[string]string
	// TemplateData is the data to use in template rendering.
	// Rendering will not take place if it is nil when InitTemplates is called.
	TemplateData interface{}
	// MetaMatcher is a function that returns true if the given resource metadata matches at
	// least one of the given names.
	// The matcher implemented using TemplatedMetaSliceMatcher can compare names to any meta field.
	MetaMatcher func(names map[string]string, meta yaml.ResourceMeta) bool
}

// Match parses the resource node's metadata and delegates matching logic to the provided
// MetaMatcher func. This allows ResourceMatchers build with TemplatedMetaMapMatcher to match
// against any field in resource metadata.
func (m *TemplatedMetaMapMatcher) Match(node *yaml.RNode) bool {
	var err error
	meta, err := node.GetMeta()
	if err != nil {
		return false
	}

	return m.MetaMatcher(m.values, meta)
}

// DefaultTemplateData sets TemplateData to the provided default values if it has not already
// been set.
func (m *TemplatedMetaMapMatcher) DefaultTemplateData(data interface{}) {
	if m.TemplateData == nil {
		m.TemplateData = data
	}
}

// InitTemplates is used to render any templates the selector's key-value pairs may contain
// before the selector is applied. It should be called exactly once per filter
// operation, before beginning match comparisons.
func (m *TemplatedMetaMapMatcher) InitTemplates() error {
	var err error
	m.values, err = templatizeMap(m.Templates, m.TemplateData)
	return errors.Wrap(err)
}

// Filter applies the matcher to a list of items, returning only those that match.
func (m *TemplatedMetaMapMatcher) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	// AndSelector or OrSelector doesn't really matter here since there is only one matcher (m).
	s := AndSelector{Matchers: []ResourceMatcher{m}, TemplateData: m.TemplateData}
	return s.Filter(items)
}

var _ ResourceTemplateMatcher = &TemplatedMetaMapMatcher{}

func templatizeSlice(values []string, data interface{}) ([]string, error) {
	if data == nil {
		return values, nil
	}
	var err error
	results := make([]string, len(values))
	for i := range values {
		results[i], err = templatize(values[i], data)
		if err != nil {
			return nil, errors.WrapPrefixf(err, "unable to render template %s", values[i])
		}
	}
	return results, nil
}

func templatizeMap(values map[string]string, data interface{}) (map[string]string, error) {
	if data == nil {
		return values, nil
	}
	var err error
	results := make(map[string]string, len(values))

	for k := range values {
		results[k], err = templatize(values[k], data)
		if err != nil {
			return nil, errors.WrapPrefixf(err, "unable to render template for %s=%s", k, values[k])
		}
	}
	return results, nil
}

// templatize renders the value as a template, using the provided data
func templatize(value string, data interface{}) (string, error) {
	t, err := template.New("kinds").Parse(value)
	if err != nil {
		return "", errors.Wrap(err)
	}
	var b bytes.Buffer
	err = t.Execute(&b, data)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return b.String(), nil
}
