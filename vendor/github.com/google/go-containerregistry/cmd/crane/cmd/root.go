// Copyright 2019 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/internal/cmd"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

const (
	use   = "crane"
	short = "Crane is a tool for managing container images"
)

var Root = New(use, short, []crane.Option{})

// New returns a top-level command for crane. This is mostly exposed
// to share code with gcrane.
func New(use, short string, options []crane.Option) *cobra.Command {
	verbose := false
	insecure := false
	ndlayers := false
	platform := &platformValue{}

	root := &cobra.Command{
		Use:               use,
		Short:             short,
		RunE:              func(cmd *cobra.Command, _ []string) error { return cmd.Usage() },
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			options = append(options, crane.WithContext(cmd.Context()))
			// TODO(jonjohnsonjr): crane.Verbose option?
			if verbose {
				logs.Debug.SetOutput(os.Stderr)
			}
			if insecure {
				options = append(options, crane.Insecure)
			}
			if ndlayers {
				options = append(options, crane.WithNondistributable())
			}
			if Version != "" {
				binary := "crane"
				if len(os.Args[0]) != 0 {
					binary = filepath.Base(os.Args[0])
				}
				options = append(options, crane.WithUserAgent(fmt.Sprintf("%s/%s", binary, Version)))
			}

			options = append(options, crane.WithPlatform(platform.platform))

			transport := remote.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: insecure, //nolint: gosec
			}

			var rt http.RoundTripper = transport

			// Add any http headers if they are set in the config file.
			cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
			if err != nil {
				logs.Debug.Printf("failed to read config file: %v", err)
			} else if len(cf.HTTPHeaders) != 0 {
				rt = &headerTransport{
					inner:       rt,
					httpHeaders: cf.HTTPHeaders,
				}
			}

			options = append(options, crane.WithTransport(rt))
		},
	}

	root.AddCommand(
		NewCmdAppend(&options),
		NewCmdAuth(options, "crane", "auth"),
		NewCmdBlob(&options),
		NewCmdCatalog(&options, "crane"),
		NewCmdConfig(&options),
		NewCmdCopy(&options),
		NewCmdDelete(&options),
		NewCmdDigest(&options),
		cmd.NewCmdEdit(&options),
		NewCmdExport(&options),
		NewCmdFlatten(&options),
		NewCmdIndex(&options),
		NewCmdList(&options),
		NewCmdManifest(&options),
		NewCmdMutate(&options),
		NewCmdOptimize(&options),
		NewCmdPull(&options),
		NewCmdPush(&options),
		NewCmdRebase(&options),
		NewCmdTag(&options),
		NewCmdValidate(&options),
		NewCmdVersion(),
		newCmdRegistry(),
	)

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logs")
	root.PersistentFlags().BoolVar(&insecure, "insecure", false, "Allow image references to be fetched without TLS")
	root.PersistentFlags().BoolVar(&ndlayers, "allow-nondistributable-artifacts", false, "Allow pushing non-distributable (foreign) layers")
	root.PersistentFlags().Var(platform, "platform", "Specifies the platform in the form os/arch[/variant][:osversion] (e.g. linux/amd64).")

	return root
}

// headerTransport sets headers on outgoing requests.
type headerTransport struct {
	httpHeaders map[string]string
	inner       http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (ht *headerTransport) RoundTrip(in *http.Request) (*http.Response, error) {
	for k, v := range ht.httpHeaders {
		if http.CanonicalHeaderKey(k) == "User-Agent" {
			// Docker sets this, which is annoying, since we're not docker.
			// We might want to revisit completely ignoring this.
			continue
		}
		in.Header.Set(k, v)
	}
	return ht.inner.RoundTrip(in)
}
