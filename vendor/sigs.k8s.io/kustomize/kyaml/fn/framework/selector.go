// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Selector matches resources.  A resource matches if and only if ALL of the Selector fields
// match the resource.  An empty Selector matches all resources.
type Selector struct {
	// Names is a list of metadata.names to match.  If empty match all names.
	// e.g. Names: ["foo", "bar"] matches if `metadata.name` is either "foo" or "bar".
	Names []string `json:"names" yaml:"names"`

	// Namespaces is a list of metadata.namespaces to match.  If empty match all namespaces.
	// e.g. Namespaces: ["foo", "bar"] matches if `metadata.namespace` is either "foo" or "bar".
	Namespaces []string `json:"namespaces" yaml:"namespaces"`

	// Kinds is a list of kinds to match.  If empty match all kinds.
	// e.g. Kinds: ["foo", "bar"] matches if `kind` is either "foo" or "bar".
	Kinds []string `json:"kinds" yaml:"kinds"`

	// APIVersions is a list of apiVersions to match.  If empty apply match all apiVersions.
	// e.g. APIVersions: ["foo/v1", "bar/v1"] matches if `apiVersion` is either "foo/v1" or "bar/v1".
	APIVersions []string `json:"apiVersions" yaml:"apiVersions"`

	// Labels is a collection of labels to match.  All labels must match exactly.
	// e.g. Labels: {"foo": "bar", "baz": "buz"] matches if BOTH "foo" and "baz" labels match.
	Labels map[string]string `json:"labels" yaml:"labels"`

	// Annotations is a collection of annotations to match.  All annotations must match exactly.
	// e.g. Annotations: {"foo": "bar", "baz": "buz"] matches if BOTH "foo" and "baz" annotations match.
	Annotations map[string]string `json:"annotations" yaml:"annotations"`

	// ResourceMatcher is an arbitrary function used to match resources.
	// Selector matches if the function returns true.
	ResourceMatcher func(*yaml.RNode) bool

	// TemplateData if present will cause the selector values to be parsed as templates
	// and rendered using TemplateData before they are used.
	TemplateData interface{}

	// FailOnEmptyMatch makes the selector return an error when no items are selected.
	FailOnEmptyMatch bool
}

// Filter implements kio.Filter, returning only those items from the list that the selector
// matches.
func (s *Selector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	andSel := AndSelector{TemplateData: s.TemplateData, FailOnEmptyMatch: s.FailOnEmptyMatch}
	if s.Names != nil {
		andSel.Matchers = append(andSel.Matchers, NameMatcher(s.Names...))
	}
	if s.Namespaces != nil {
		andSel.Matchers = append(andSel.Matchers, NamespaceMatcher(s.Namespaces...))
	}
	if s.Kinds != nil {
		andSel.Matchers = append(andSel.Matchers, KindMatcher(s.Kinds...))
	}
	if s.APIVersions != nil {
		andSel.Matchers = append(andSel.Matchers, APIVersionMatcher(s.APIVersions...))
	}
	if s.Labels != nil {
		andSel.Matchers = append(andSel.Matchers, LabelMatcher(s.Labels))
	}
	if s.Annotations != nil {
		andSel.Matchers = append(andSel.Matchers, AnnotationMatcher(s.Annotations))
	}
	if s.ResourceMatcher != nil {
		andSel.Matchers = append(andSel.Matchers, ResourceMatcherFunc(s.ResourceMatcher))
	}
	return andSel.Filter(items)
}

// MatchAll is a shorthand for building an AndSelector from a list of ResourceMatchers.
func MatchAll(matchers ...ResourceMatcher) *AndSelector {
	return &AndSelector{Matchers: matchers}
}

// MatchAny is a shorthand for building an OrSelector from a list of ResourceMatchers.
func MatchAny(matchers ...ResourceMatcher) *OrSelector {
	return &OrSelector{Matchers: matchers}
}

// OrSelector is a kio.Filter that selects resources when that match at least one of its embedded
// matchers.
type OrSelector struct {
	// Matchers is the list of ResourceMatchers to try on the input resources.
	Matchers []ResourceMatcher
	// TemplateData, if present, is used to initialize any matchers that implement
	// ResourceTemplateMatcher.
	TemplateData interface{}
	// FailOnEmptyMatch makes the selector return an error when no items are selected.
	FailOnEmptyMatch bool
}

// Match implements ResourceMatcher so that OrSelectors can be composed
func (s *OrSelector) Match(item *yaml.RNode) bool {
	for _, matcher := range s.Matchers {
		if matcher.Match(item) {
			return true
		}
	}
	return false
}

// Filter implements kio.Filter, returning only those items from the list that the selector
// matches.
func (s *OrSelector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if err := initMatcherTemplates(s.Matchers, s.TemplateData); err != nil {
		return nil, err
	}

	var selectedItems []*yaml.RNode
	for i := range items {
		for _, matcher := range s.Matchers {
			if matcher.Match(items[i]) {
				selectedItems = append(selectedItems, items[i])
				break
			}
		}
	}
	if s.FailOnEmptyMatch && len(selectedItems) == 0 {
		return nil, errors.Errorf("selector did not select any items")
	}
	return selectedItems, nil
}

// DefaultTemplateData makes OrSelector a ResourceTemplateMatcher.
// Although it does not contain templates itself, this allows it to support ResourceTemplateMatchers
// when being used as a matcher itself.
func (s *OrSelector) DefaultTemplateData(data interface{}) {
	if s.TemplateData == nil {
		s.TemplateData = data
	}
}

func (s *OrSelector) InitTemplates() error {
	return initMatcherTemplates(s.Matchers, s.TemplateData)
}

func initMatcherTemplates(matchers []ResourceMatcher, data interface{}) error {
	for _, matcher := range matchers {
		if tm, ok := matcher.(ResourceTemplateMatcher); ok {
			tm.DefaultTemplateData(data)
			if err := tm.InitTemplates(); err != nil {
				return err
			}
		}
	}
	return nil
}

var _ ResourceTemplateMatcher = &OrSelector{}

// AndSelector is a kio.Filter that selects resources when that match all of its embedded
// matchers.
type AndSelector struct {
	// Matchers is the list of ResourceMatchers to try on the input resources.
	Matchers []ResourceMatcher
	// TemplateData, if present, is used to initialize any matchers that implement
	// ResourceTemplateMatcher.
	TemplateData interface{}
	// FailOnEmptyMatch makes the selector return an error when no items are selected.
	FailOnEmptyMatch bool
}

// Match implements ResourceMatcher so that AndSelectors can be composed
func (s *AndSelector) Match(item *yaml.RNode) bool {
	for _, matcher := range s.Matchers {
		if !matcher.Match(item) {
			return false
		}
	}
	return true
}

// Filter implements kio.Filter, returning only those items from the list that the selector
// matches.
func (s *AndSelector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if err := initMatcherTemplates(s.Matchers, s.TemplateData); err != nil {
		return nil, err
	}
	var selectedItems []*yaml.RNode
	for i := range items {
		isSelected := true
		for _, matcher := range s.Matchers {
			if !matcher.Match(items[i]) {
				isSelected = false
				break
			}
		}
		if isSelected {
			selectedItems = append(selectedItems, items[i])
		}
	}
	if s.FailOnEmptyMatch && len(selectedItems) == 0 {
		return nil, errors.Errorf("selector did not select any items")
	}
	return selectedItems, nil
}

// DefaultTemplateData makes AndSelector a ResourceTemplateMatcher.
// Although it does not contain templates itself, this allows it to support ResourceTemplateMatchers
// when being used as a matcher itself.
func (s *AndSelector) DefaultTemplateData(data interface{}) {
	if s.TemplateData == nil {
		s.TemplateData = data
	}
}

func (s *AndSelector) InitTemplates() error {
	return initMatcherTemplates(s.Matchers, s.TemplateData)
}

var _ ResourceTemplateMatcher = &AndSelector{}
