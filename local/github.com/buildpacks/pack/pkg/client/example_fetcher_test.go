//go:build !windows && example

package client_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
)

// This example shows how to replace the image fetcher component
func Example_fetcher() {
	// create a context object
	context := context.Background()

	// initialize a pack client
	pack, err := client.NewClient(client.WithFetcher(&fetcher{}))
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
	_ = pack.Build(context, buildOpts)

	// Output: custom fetcher called
}

var _ client.ImageFetcher = (*fetcher)(nil)

type fetcher struct{}

func (f *fetcher) Fetch(_ context.Context, imageName string, _ image.FetchOptions) (imgutil.Image, error) {
	fmt.Println("custom fetcher called")
	return nil, errors.New("not implemented")
}

func (f *fetcher) CheckReadAccess(_ string, _ image.FetchOptions) bool {
	return true
}
