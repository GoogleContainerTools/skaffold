// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// Package framework contains a framework for writing functions in Go.  The function specification
// is defined at: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
//
// Functions are executables that generate, modify, delete or validate Kubernetes resources.
// They are often used used to implement abstractions ("kind: JavaSpringBoot") and
// cross-cutting logic ("kind: SidecarInjector").
//
// Functions may be run as standalone executables or invoked as part of an orchestrated
// pipeline (e.g. kustomize).
//
// Example function implementation using framework.SimpleProcessor with a struct input
//
//	import (
//		"sigs.k8s.io/kustomize/kyaml/errors"
//		"sigs.k8s.io/kustomize/kyaml/fn/framework"
//		"sigs.k8s.io/kustomize/kyaml/kio"
//		"sigs.k8s.io/kustomize/kyaml/yaml"
//	)
//
//	type Spec struct {
//		Value string `yaml:"value,omitempty"`
//	}
//	type Example struct {
//		Spec Spec `yaml:"spec,omitempty"`
//	}
//
//	func runFunction(rlSource *kio.ByteReadWriter) error {
//		functionConfig := &Example{}
//
//		fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
//			for i := range items {
//				// modify the items...
//			}
//			return items, nil
//		}
//
//		p := framework.SimpleProcessor{Config: functionConfig, Filter: kio.FilterFunc(fn)}
//		err := framework.Execute(p, rlSource)
//		return errors.Wrap(err)
//	}
//
// Architecture
//
// Functions modify a slice of resources (ResourceList.Items) which are read as input and written
// as output.  The function itself may be configured through a functionConfig
// (ResourceList.FunctionConfig).
//
// Example function input:
//
//    kind: ResourceList
//    items:
//    - kind: Deployment
//      ...
//    - kind: Service
//      ....
//    functionConfig:
//      kind: Example
//      spec:
//        value: foo
//
// The functionConfig may be specified declaratively and run with
//
//	kustomize fn run DIR/
//
// Declarative function declaration:
//
//    kind: Example
//    metadata:
//      annotations:
//        # run the function by creating this container and providing this
//        # Example as the functionConfig
//        config.kubernetes.io/function: |
//          container:
//            image: image/containing/function:impl
//    spec:
//      value: foo
//
// The framework takes care of serializing and deserializing the ResourceList.
//
// Generated ResourceList.functionConfig -- ConfigMaps
// Functions may also be specified imperatively and run using:
//
//	kustomize fn run DIR/ --image image/containing/function:impl -- value=foo
//
// When run imperatively, a ConfigMap is generated for the functionConfig, and the command
// arguments are set as ConfigMap data entries.
//
//    kind: ConfigMap
//    data:
//      value: foo
//
// To write a function that can be run imperatively on the commandline, have it take a
// ConfigMap as its functionConfig.
//
// Mutator and Generator Functions
//
// Functions may add, delete or modify resources by modifying the ResourceList.Items slice.
//
// Validator Functions
//
// A function may emit validation results by setting the ResourceList.Result
//
// Configuring Functions
//
// Functions may be configured through a functionConfig (i.e. a client-side custom resource),
// or through flags (which the framework parses from a ConfigMap provided as input).
//
// Functions may also access environment variables set by the caller.
//
// Building a container image for the function
//
// The go program may be built into a container and run as a function.  The framework
// can be used to generate a Dockerfile to build the function container.
//
//   # create the ./Dockerfile for the container
//   $ go run ./main.go gen ./
//
//   # build the function's container
//   $ docker build . -t gcr.io/my-project/my-image:my-version
package framework
