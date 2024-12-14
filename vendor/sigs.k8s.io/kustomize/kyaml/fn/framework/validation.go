// Copyright 2022 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/resid"
	k8syaml "sigs.k8s.io/yaml"
)

// SchemaFromFunctionDefinition extracts the schema for a particular GVK from the provided KRMFunctionDefinition
// Since the relevant fields of KRMFunctionDefinition exactly match the ones in CustomResourceDefinition,
// this helper can also load CRDs (e.g. produced by KubeBuilder) transparently.
func SchemaFromFunctionDefinition(gvk resid.Gvk, data string) (*spec.Schema, error) {
	var def KRMFunctionDefinition
	// need to use sigs yaml because spec.Schema type only has json tags
	if err := k8syaml.Unmarshal([]byte(data), &def); err != nil {
		return nil, errors.WrapPrefixf(err, "unmarshalling %s", FunctionDefinitionKind)
	}
	var foundGVKs []*resid.Gvk
	var schema *spec.Schema
	for i, version := range def.Spec.Versions {
		versionGVK := resid.Gvk{Group: def.Spec.Group, Kind: def.Spec.Names.Kind, Version: version.Name}
		if gvk.Equals(versionGVK) {
			schema = def.Spec.Versions[i].Schema.OpenAPIV3Schema
			break
		}
		foundGVKs = append(foundGVKs, &versionGVK)
	}
	if schema == nil {
		return nil, errors.Errorf("%s does not define %s (defines: %s)", FunctionDefinitionKind, gvk, foundGVKs)
	}
	return schema, nil
}
