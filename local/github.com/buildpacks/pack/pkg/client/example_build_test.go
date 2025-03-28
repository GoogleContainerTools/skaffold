//go:build !windows && example

package client_test

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/buildpacks/pack/pkg/client"
)

// This example shows the basic usage of the package: Create a client,
// call a configuration object, call the client's Build function.
func Example_build() {
	// create a context object
	context := context.Background()

	// initialize a pack client
	pack, err := client.NewClient()
	if err != nil {
		panic(err)
	}

	// replace this with the location of a sample application
	// For a list of prepared samples see the 'apps' folder at
	// https://github.com/buildpacks/samples.
	appPath := filepath.Join("testdata", "some-app")

	// initialize our options
	buildOpts := client.BuildOptions{
		Image:        "pack-lib-test-image:0.0.1",
		Builder:      "cnbs/sample-builder:bionic",
		AppPath:      appPath,
		TrustBuilder: func(string) bool { return true },
	}

	// build an image
	err = pack.Build(context, buildOpts)
	if err != nil {
		panic(err)
	}

	fmt.Println("build completed")
	// Output: build completed
}
