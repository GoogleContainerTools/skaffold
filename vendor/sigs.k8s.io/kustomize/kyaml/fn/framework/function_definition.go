// Copyright 2022 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const FunctionDefinitionKind = "KRMFunctionDefinition"
const FunctionDefinitionGroupVersion = "config.kubernetes.io/v1alpha1"

// KRMFunctionDefinition is metadata that defines a KRM function the same way a CRD defines a custom resource.
// https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/2906-kustomize-function-catalog#function-metadata-schema
type KRMFunctionDefinition struct {
	// APIVersion and Kind of the object. Must be config.kubernetes.io/v1alpha1 and KRMFunctionDefinition respectively.
	yaml.TypeMeta `yaml:",inline" json:",inline"`
	// Standard KRM object metadata
	yaml.ObjectMeta `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	// Spec contains the properties of the KRM function this object defines.
	Spec KrmFunctionDefinitionSpec `yaml:"spec" json:"spec"`
}

type KrmFunctionDefinitionSpec struct {
	//
	// The following fields are shared with CustomResourceDefinition.
	//
	// Group is the API group of the defined KRM function.
	Group string `yaml:"group" json:"group"`
	// Names specify the resource and kind names for the KRM function.
	Names KRMFunctionNames `yaml:"names" json:"names"`
	// Versions is the list of all API versions of the defined KRM function.
	Versions []KRMFunctionVersion `yaml:"versions" json:"versions"`

	//
	// The following fields are custom to KRMFunctionDefinition
	//
	// Description briefly describes the KRM function.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Publisher is the entity (e.g. organization) that produced and owns this KRM function.
	Publisher string `yaml:"publisher,omitempty" json:"publisher,omitempty"`
	// Home is a URI pointing the home page of the KRM function.
	Home string `yaml:"home,omitempty" json:"home,omitempty"`
	// Maintainers lists the individual maintainers of the KRM function.
	Maintainers []string `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	// Tags are keywords describing the function. e.g. mutator, validator, generator, prefix, GCP.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

type KRMFunctionVersion struct {
	//
	// The following fields are shared with CustomResourceDefinition.
	//
	// Name is the version name, e.g. “v1”, “v2beta1”, etc.
	Name string `yaml:"name" json:"name"`
	// Schema describes the schema of this version of the KRM function.
	// This can be used for validation, pruning, and/or defaulting.
	Schema *KRMFunctionValidation `yaml:"schema,omitempty" json:"schema,omitempty"`

	//
	// The following fields are custom to KRMFunctionDefinition
	//
	// Idempotent indicates whether the function can be re-run multiple times without changing the result.
	Idempotent bool `yaml:"idempotent,omitempty" json:"idempotent,omitempty"`
	// Usage is URI pointing to a README.md that describe the details of how to use the KRM function.
	// It should at least cover what the function does and should give a detailed explanation about each
	// field used to configure it.
	Usage string `yaml:"usage,omitempty" json:"usage,omitempty"`
	// A list of URIs that point to README.md files. Each README.md should cover an example.
	// It should at least cover how to get input resources, how to run it and what is the expected
	// output.
	Examples []string `yaml:"examples,omitempty" json:"examples,omitempty"`
	// License is the name of the license covering the function.
	License string `yaml:"license,omitempty" json:"license,omitempty"`
	// The maintainers for this version of the function, if different from the primary maintainers.
	Maintainers []string `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	// The runtime information describing how to execute this function.
	Runtime runtimeutil.FunctionSpec `yaml:"runtime" json:"runtime"`
}

type KRMFunctionValidation struct {
	// OpenAPIV3Schema is the OpenAPI v3 schema for an instance of the KRM function.
	OpenAPIV3Schema *spec.Schema `yaml:"openAPIV3Schema,omitempty" json:"openAPIV3Schema,omitempty"` //nolint: tagliatelle
}

type KRMFunctionNames struct {
	// Kind is the kind of the defined KRM Function. It is normally CamelCase and singular.
	Kind string `yaml:"kind" json:"kind"`
}
