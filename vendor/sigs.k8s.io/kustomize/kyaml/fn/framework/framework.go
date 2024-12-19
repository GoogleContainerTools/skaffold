// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	goerrors "errors"
	"os"

	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceList is a Kubernetes list type used as the primary data interchange format
// in the Configuration Functions Specification:
// https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
// This framework facilitates building functions that receive and emit ResourceLists,
// as required by the specification.
type ResourceList struct {
	// Items is the ResourceList.items input and output value.
	//
	// e.g. given the function input:
	//
	//    kind: ResourceList
	//    items:
	//    - kind: Deployment
	//      ...
	//    - kind: Service
	//      ...
	//
	// Items will be a slice containing the Deployment and Service resources
	// Mutating functions will alter this field during processing.
	// This field is required.
	Items []*yaml.RNode `yaml:"items" json:"items"`

	// FunctionConfig is the ResourceList.functionConfig input value.
	//
	// e.g. given the input:
	//
	//    kind: ResourceList
	//    functionConfig:
	//      kind: Example
	//      spec:
	//        foo: var
	//
	// FunctionConfig will contain the RNodes for the Example:
	//      kind: Example
	//      spec:
	//        foo: var
	FunctionConfig *yaml.RNode `yaml:"functionConfig,omitempty" json:"functionConfig,omitempty"`

	// Results is ResourceList.results output value.
	// Validating functions can optionally use this field to communicate structured
	// validation error data to downstream functions.
	Results Results `yaml:"results,omitempty" json:"results,omitempty"`
}

// ResourceListProcessor is implemented by configuration functions built with this framework
// to conform to the Configuration Functions Specification:
// https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
// To invoke a processor, pass it to framework.Execute, which will also handle ResourceList IO.
//
// This framework provides several ready-to-use ResourceListProcessors, including
// SimpleProcessor, VersionedAPIProcessor and TemplateProcessor.
// You can also build your own by implementing this interface.
type ResourceListProcessor interface {
	Process(rl *ResourceList) error
}

// ResourceListProcessorFunc converts a compatible function to a ResourceListProcessor.
type ResourceListProcessorFunc func(rl *ResourceList) error

// Process makes ResourceListProcessorFunc implement the ResourceListProcessor interface.
func (p ResourceListProcessorFunc) Process(rl *ResourceList) error {
	return p(rl)
}

// Defaulter is implemented by APIs to have Default invoked.
// The standard application is to create a type to hold your FunctionConfig data, and
// implement Defaulter on that type. All of the framework's processors will invoke Default()
// on your type after unmarshalling the FunctionConfig data into it.
type Defaulter interface {
	Default() error
}

// Validator is implemented by APIs to have Validate invoked.
// The standard application is to create a type to hold your FunctionConfig data, and
// implement Validator on that type. All of the framework's processors will invoke Validate()
// on your type after unmarshalling the FunctionConfig data into it.
type Validator interface {
	Validate() error
}

// ValidationSchemaProvider is implemented by APIs to have the openapi schema provided by Schema()
// used to validate the input functionConfig before it is parsed into the API's struct.
// Use this with framework.SchemaFromFunctionDefinition to load the schema out of a KRMFunctionDefinition
// or CRD (e.g. one generated with KubeBuilder).
//
// func (t MyType) Schema() (*spec.Schema, error) {
//	 schema, err := framework.SchemaFromFunctionDefinition(resid.NewGvk("example.com", "v1", "MyType"), MyTypeDef)
//	 return schema, errors.WrapPrefixf(err, "parsing MyType schema")
// }
type ValidationSchemaProvider interface {
	Schema() (*spec.Schema, error)
}

// Execute is the entrypoint for invoking configuration functions built with this framework
// from code. See framework/command#Build for a Cobra-based command-line equivalent.
// Execute reads a ResourceList from the given source, passes it to a ResourceListProcessor,
// and then writes the result to the target.
// STDIN and STDOUT will be used if no reader or writer respectively is provided.
func Execute(p ResourceListProcessor, rlSource *kio.ByteReadWriter) error {
	// Prepare the resource list source
	if rlSource == nil {
		rlSource = &kio.ByteReadWriter{KeepReaderAnnotations: true}
	}
	if rlSource.Reader == nil {
		rlSource.Reader = os.Stdin
	}
	if rlSource.Writer == nil {
		rlSource.Writer = os.Stdout
	}

	// Read the input
	rl := ResourceList{}
	var err error
	if rl.Items, err = rlSource.Read(); err != nil {
		return errors.WrapPrefixf(err, "failed to read ResourceList input")
	}
	rl.FunctionConfig = rlSource.FunctionConfig

	// We store the original
	nodeAnnos, err := kio.PreprocessResourcesForInternalAnnotationMigration(rl.Items)
	if err != nil {
		return err
	}

	retErr := p.Process(&rl)

	// If either the internal annotations for path, index, and id OR the legacy
	// annotations for path, index, and id are changed, we have to update the other.
	err = kio.ReconcileInternalAnnotations(rl.Items, nodeAnnos)
	if err != nil {
		return err
	}

	// Write the results
	// Set the ResourceList.results for validating functions
	if len(rl.Results) > 0 {
		b, err := yaml.Marshal(rl.Results)
		if err != nil {
			return errors.Wrap(err)
		}
		y, err := yaml.Parse(string(b))
		if err != nil {
			return errors.Wrap(err)
		}
		rlSource.Results = y
	}
	if err := rlSource.Write(rl.Items); err != nil {
		return err
	}

	return retErr
}

// Filter executes the given kio.Filter and replaces the ResourceList's items with the result.
// This can be used to help implement ResourceListProcessors. See SimpleProcessor for example.
//
// Filters that return a Result as error will store the result in the ResourceList
// and continue processing instead of erroring out.
func (rl *ResourceList) Filter(api kio.Filter) error {
	if api == nil {
		return errors.Errorf("ResourceList cannot run apply nil filter")
	}
	var err error
	rl.Items, err = api.Filter(rl.Items)
	if err != nil {
		var r Results
		if goerrors.As(err, &r) {
			rl.Results = append(rl.Results, r...)
			return nil
		}
		return errors.Wrap(err)
	}
	return nil
}
