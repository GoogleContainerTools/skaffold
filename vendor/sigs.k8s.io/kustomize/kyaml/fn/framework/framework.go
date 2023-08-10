// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"os"

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
	FunctionConfig *yaml.RNode `yaml:"functionConfig" json:"functionConfig"`

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
	Items []*yaml.RNode `yaml:"items" json:"items"`

	// Result is ResourceList.result output value.
	// Validating functions can optionally use this field to communicate structured
	// validation error data to downstream functions.
	Result *Result `yaml:"results" json:"results"`
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
		return errors.Wrap(err)
	}
	rl.FunctionConfig = rlSource.FunctionConfig

	retErr := p.Process(&rl)

	// Write the results
	// Set the ResourceList.results for validating functions
	if rl.Result != nil && len(rl.Result.Items) > 0 {
		b, err := yaml.Marshal(rl.Result)
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
func (rl *ResourceList) Filter(api kio.Filter) error {
	var err error
	rl.Items, err = api.Filter(rl.Items)
	return errors.Wrap(err)
}
