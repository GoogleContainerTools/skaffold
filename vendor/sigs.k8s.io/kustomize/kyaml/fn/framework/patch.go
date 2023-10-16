// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// ResourcePatchTemplate applies a patch to a collection of resources
type ResourcePatchTemplate struct {
	// Templates is a function that returns a list of templates to render into one or more patches.
	Templates TemplatesFunc

	// Selector targets the rendered patches to specific resources. If no Selector is provided,
	// all resources will be patched.
	//
	// Although any Filter can be used, this framework provides several especially for Selector use:
	// framework.Selector, framework.AndSelector, framework.OrSelector. You can also use any of the
	// framework's ResourceMatchers here directly.
	Selector kio.Filter

	// TemplateData is the data to use when rendering the templates provided by the Templates field.
	TemplateData interface{}
}

// DefaultTemplateData sets TemplateData to the provided default values if it has not already
// been set.
func (t *ResourcePatchTemplate) DefaultTemplateData(data interface{}) {
	if t.TemplateData == nil {
		t.TemplateData = data
	}
}

// Filter applies the ResourcePatchTemplate to the appropriate resources in the input.
// First, it applies the Selector to identify target resources. Then, it renders the Templates
// into patches using TemplateData. Finally, it identifies applies the patch to each resource.
func (t ResourcePatchTemplate) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error
	target := items
	if t.Selector != nil {
		target, err = t.Selector.Filter(items)
		if err != nil {
			return nil, err
		}
	}
	if len(target) == 0 {
		// nothing to do
		return items, nil
	}

	if err := t.apply(target); err != nil {
		return nil, errors.Wrap(err)
	}
	return items, nil
}

func (t *ResourcePatchTemplate) apply(matches []*yaml.RNode) error {
	templates, err := t.Templates()
	if err != nil {
		return errors.Wrap(err)
	}
	var patches []*yaml.RNode
	for i := range templates {
		newP, err := renderPatches(templates[i], t.TemplateData)
		if err != nil {
			return errors.Wrap(err)
		}
		patches = append(patches, newP...)
	}

	// apply the patches to the matching resources
	for j := range matches {
		for i := range patches {
			matches[j], err = merge2.Merge(patches[i], matches[j], yaml.MergeOptions{})
			if err != nil {
				return errors.WrapPrefixf(err, "failed to apply templated patch")
			}
		}
	}
	return nil
}

// ContainerPatchTemplate defines a patch to be applied to containers
type ContainerPatchTemplate struct {
	// Templates is a function that returns a list of templates to render into one or more
	// patches that apply at the container level. For example, "name", "env" and "image" would be
	// top-level fields in container patches.
	Templates TemplatesFunc

	// Selector targets the rendered patches to containers within specific resources.
	// If no Selector is provided, all resources with containers will be patched (subject to
	// ContainerMatcher, if provided).
	//
	// Although any Filter can be used, this framework provides several especially for Selector use:
	// framework.Selector, framework.AndSelector, framework.OrSelector. You can also use any of the
	// framework's ResourceMatchers here directly.
	Selector kio.Filter

	// TemplateData is the data to use when rendering the templates provided by the Templates field.
	TemplateData interface{}

	// ContainerMatcher targets the rendered patch to only those containers it matches.
	// For example, it can be used with ContainerNameMatcher to patch only containers with
	// specific names. If no ContainerMatcher is provided, all containers will be patched.
	//
	// The node passed to ContainerMatcher will be container-level, not a full resource node.
	// For example, "name", "env" and "image" would be top level fields.
	// To filter based on resource-level context, use the Selector field.
	ContainerMatcher func(node *yaml.RNode) bool
}

// DefaultTemplateData sets TemplateData to the provided default values if it has not already
// been set.
func (cpt *ContainerPatchTemplate) DefaultTemplateData(data interface{}) {
	if cpt.TemplateData == nil {
		cpt.TemplateData = data
	}
}

// Filter applies the ContainerPatchTemplate to the appropriate resources in the input.
// First, it applies the Selector to identify target resources. Then, it renders the Templates
// into patches using TemplateData. Finally, it identifies target containers and applies the
// patches.
func (cpt ContainerPatchTemplate) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error
	target := items
	if cpt.Selector != nil {
		target, err = cpt.Selector.Filter(items)
		if err != nil {
			return nil, err
		}
	}
	if len(target) == 0 {
		// nothing to do
		return items, nil
	}

	if err := cpt.apply(target); err != nil {
		return nil, err
	}

	return items, nil
}

// PatchContainers applies the patch to each matching container in each resource.
func (cpt ContainerPatchTemplate) apply(matches []*yaml.RNode) error {
	templates, err := cpt.Templates()
	if err != nil {
		return errors.Wrap(err)
	}
	var patches []*yaml.RNode
	for i := range templates {
		newP, err := renderPatches(templates[i], cpt.TemplateData)
		if err != nil {
			return errors.Wrap(err)
		}
		patches = append(patches, newP...)
	}

	for i := range matches {
		// TODO(knverey): Make this work for more Kinds and expose the helper for doing so.
		containers, err := matches[i].Pipe(yaml.Lookup("spec", "template", "spec", "containers"))
		if err != nil {
			return errors.Wrap(err)
		}
		if containers == nil {
			continue
		}
		err = containers.VisitElements(func(node *yaml.RNode) error {
			if cpt.ContainerMatcher != nil && !cpt.ContainerMatcher(node) {
				return nil
			}
			for j := range patches {
				merger := walk.Walker{
					Sources:      []*yaml.RNode{node, patches[j]}, // dest, src
					Visitor:      merge2.Merger{},
					MergeOptions: yaml.MergeOptions{},
					Schema: openapi.SchemaForResourceType(yaml.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					}).Lookup("spec", "containers").Elements(),
				}
				_, err = merger.Walk()
				if err != nil {
					return errors.WrapPrefixf(err, "failed to apply templated patch")
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err)
		}
	}
	return nil
}

func renderPatches(t *template.Template, data interface{}) ([]*yaml.RNode, error) {
	// render the patches
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return nil, errors.WrapPrefixf(err, "failed to render patch template %v", t.DefinedTemplates())
	}

	// parse the patches into RNodes
	var nodes []*yaml.RNode
	for _, s := range strings.Split(b.String(), "\n---\n") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		r := &kio.ByteReader{Reader: bytes.NewBufferString(s), OmitReaderAnnotations: true}
		newNodes, err := r.Read()
		if err != nil {
			return nil, errors.WrapPrefixf(err,
				"failed to parse rendered patch template into a resource:\n%s\n", addLineNumbers(s))
		}
		if err := yaml.ErrorIfAnyInvalidAndNonNull(yaml.MappingNode, newNodes...); err != nil {
			return nil, errors.WrapPrefixf(err,
				"failed to parse rendered patch template into a resource:\n%s\n", addLineNumbers(s))
		}
		nodes = append(nodes, newNodes...)
	}
	return nodes, nil
}

func addLineNumbers(s string) string {
	lines := strings.Split(s, "\n")
	for j := range lines {
		lines[j] = fmt.Sprintf("%03d %s", j+1, lines[j])
	}
	return strings.Join(lines, "\n")
}
