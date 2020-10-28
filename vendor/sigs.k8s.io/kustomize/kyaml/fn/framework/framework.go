// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceList reads the function input and writes the function output.
//
// Adheres to the spec: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
type ResourceList struct {
	// FunctionConfig is the ResourceList.functionConfig input value.  If FunctionConfig
	// is set to a value such as a struct or map[string]interface{} before ResourceList.Read()
	// is called, then the functionConfig will be parsed into that value.
	// If it is nil, the functionConfig will be set to a map[string]interface{}
	// before it is parsed.
	//
	// e.g. given the function input:
	//
	//    kind: ResourceList
	//    functionConfig:
	//      kind: Example
	//      spec:
	//        foo: var
	//
	// FunctionConfig will contain the Example unmarshalled into its value.
	FunctionConfig interface{}

	// Items is the ResourceList.items input and output value.  Items will be set by
	// ResourceList.Read() and written by ResourceList.Write().
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
	Items []*yaml.RNode

	// Result is ResourceList.result output value.  Result will be written by
	// ResourceList.Write()
	Result *Result

	// Flags are an optional set of flags to parse the ResourceList.functionConfig.data.
	// If non-nil, ResourceList.Read() will set the flag value for each flag name matching
	// a ResourceList.functionConfig.data map entry.
	//
	// e.g. given the function input:
	//
	//    kind: ResourceList
	//    functionConfig:
	//      data:
	//        foo: bar
	//        a: b
	//
	// The flags --a=b and --foo=bar will be set in Flags.
	Flags *pflag.FlagSet

	// Reader is used to read the function input (ResourceList).
	// Defaults to os.Stdin.
	Reader io.Reader

	// Writer is used to write the function output (ResourceList)
	// Defaults to os.Stdout.
	Writer io.Writer

	// rw reads function input and writes function output
	rw *kio.ByteReadWriter
}

// Read reads the ResourceList
func (r *ResourceList) Read() error {
	if r.Reader == nil {
		r.Reader = os.Stdin
	}
	if r.Writer == nil {
		r.Writer = os.Stdout
	}
	r.rw = &kio.ByteReadWriter{
		Reader:                r.Reader,
		Writer:                r.Writer,
		KeepReaderAnnotations: true,
	}

	var err error
	r.Items, err = r.rw.Read()
	if err != nil {
		return errors.Wrap(err)
	}

	// parse the functionConfig
	return func() error {
		if r.rw.FunctionConfig == nil {
			// no function config exists
			return nil
		}
		if r.FunctionConfig == nil {
			// set directly from r.rw
			r.FunctionConfig = r.rw.FunctionConfig
		} else {
			// unmarshal the functionConfig into the provided value
			err := yaml.Unmarshal([]byte(r.rw.FunctionConfig.MustString()), r.FunctionConfig)
			if err != nil {
				return errors.Wrap(err)
			}
		}

		// set the functionConfig values as flags so they are easy to access
		if r.Flags == nil || !r.Flags.HasFlags() {
			return nil
		}
		// flags are always set from the "data" field
		data, err := r.rw.FunctionConfig.Pipe(yaml.Lookup("data"))
		if err != nil || data == nil {
			return err
		}
		return data.VisitFields(func(node *yaml.MapNode) error {
			f := r.Flags.Lookup(node.Key.YNode().Value)
			if f == nil {
				return nil
			}
			return f.Value.Set(node.Value.YNode().Value)
		})
	}()
}

// Write writes the ResourceList
func (r *ResourceList) Write() error {
	// set the ResourceList.results for validating functions
	if r.Result != nil {
		if len(r.Result.Items) > 0 {
			b, err := yaml.Marshal(r.Result)
			if err != nil {
				return errors.Wrap(err)
			}
			y, err := yaml.Parse(string(b))
			if err != nil {
				return errors.Wrap(err)
			}
			r.rw.Results = y
		}
	}

	// write the results
	return r.rw.Write(r.Items)
}

// Command returns a cobra.Command to run a function.
//
// The cobra.Command will use the provided ResourceList to Read() the input,
// run the provided function, and then Write() the output.
//
// The returned cobra.Command will have a "gen" subcommand which can be used to generate
// a Dockerfile to build the function into a container image
//
//		go run main.go gen DIR/
func Command(resourceList *ResourceList, function Function) cobra.Command {
	cmd := cobra.Command{}
	AddGenerateDockerfile(&cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := execute(resourceList, function, cmd)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%v", err)
		}
		return err
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

// AddGenerateDockerfile adds a "gen" subcommand to create a Dockerfile for building
// the function as a container.
func AddGenerateDockerfile(cmd *cobra.Command) {
	gen := &cobra.Command{
		Use:  "gen",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ioutil.WriteFile(filepath.Join(args[0], "Dockerfile"), []byte(`FROM golang:1.13-stretch
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY . .
RUN go build -v -o /usr/local/bin/function ./

FROM alpine:latest
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
CMD ["function"]
`), 0600)
		},
	}
	cmd.AddCommand(gen)
}

func execute(rl *ResourceList, function Function, cmd *cobra.Command) error {
	rl.Reader = cmd.InOrStdin()
	rl.Writer = cmd.OutOrStdout()
	rl.Flags = cmd.Flags()

	if err := rl.Read(); err != nil {
		return err
	}

	retErr := function()

	if err := rl.Write(); err != nil {
		return err
	}

	return retErr
}
