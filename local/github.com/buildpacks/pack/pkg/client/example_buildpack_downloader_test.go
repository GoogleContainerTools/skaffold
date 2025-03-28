//go:build !windows && example

package client_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
)

// This example shows how to replace the buildpack downloader component
func Example_buildpack_downloader() {
	// create a context object
	context := context.Background()

	// initialize a pack client
	pack, err := client.NewClient(client.WithBuildpackDownloader(&bpDownloader{}))
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
		Buildpacks:   []string{"some-buildpack:1.2.3"},
		TrustBuilder: func(string) bool { return true },
	}

	// build an image
	_ = pack.Build(context, buildOpts)

	// Output: custom buildpack downloader called
}

var _ client.BuildpackDownloader = (*bpDownloader)(nil)

type bpDownloader struct{}

func (f *bpDownloader) Download(ctx context.Context, buildpackURI string, opts buildpack.DownloadOptions) (buildpack.BuildModule, []buildpack.BuildModule, error) {
	fmt.Println("custom buildpack downloader called")
	return nil, nil, errors.New("not implemented")
}
