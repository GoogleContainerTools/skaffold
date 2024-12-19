// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceTemplate generates resources from templates.
type ResourceTemplate struct {
	// Templates provides a list of templates to render into one or more resources.
	Templates TemplateParser

	// TemplateData is the data to use when rendering the templates provided by the Templates field.
	TemplateData interface{}
}

type TemplateParser interface {
	Parse() ([]*template.Template, error)
}

type TemplateParserFunc func() ([]*template.Template, error)

func (s TemplateParserFunc) Parse() ([]*template.Template, error) {
	return s()
}

// DefaultTemplateData sets TemplateData to the provided default values if it has not already
// been set.
func (rt *ResourceTemplate) DefaultTemplateData(data interface{}) {
	if rt.TemplateData == nil {
		rt.TemplateData = data
	}
}

// Render renders the Templates into resource nodes using TemplateData.
func (rt *ResourceTemplate) Render() ([]*yaml.RNode, error) {
	var items []*yaml.RNode

	if rt.Templates == nil {
		return items, nil
	}

	templates, err := rt.Templates.Parse()
	if err != nil {
		return nil, errors.WrapPrefixf(err, "failed to retrieve ResourceTemplates")
	}

	for i := range templates {
		newItems, err := rt.doTemplate(templates[i])
		if err != nil {
			return nil, err
		}
		items = append(items, newItems...)
	}
	return items, nil
}

func (rt *ResourceTemplate) doTemplate(t *template.Template) ([]*yaml.RNode, error) {
	// invoke the template
	var b bytes.Buffer
	err := t.Execute(&b, rt.TemplateData)
	if err != nil {
		return nil, errors.WrapPrefixf(err, "failed to render template %v", t.DefinedTemplates())
	}
	var items []*yaml.RNode

	// split the resources so the error messaging is better
	for _, s := range strings.Split(b.String(), "\n---\n") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		newItems, err := (&kio.ByteReader{Reader: bytes.NewBufferString(s)}).Read()
		if err != nil {
			return nil, errors.WrapPrefixf(err,
				"failed to parse rendered template into a resource:\n%s\n", addLineNumbers(s))
		}

		items = append(items, newItems...)
	}
	return items, nil
}
