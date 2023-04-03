// Copyright 2021 Google LLC All Rights Reserved.
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

package commands

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/ko/internal/sbom"
	"github.com/sigstore/cosign/pkg/oci/signed"
	"github.com/spf13/cobra"
)

// addDeps augments our CLI surface with deps.
func addDeps(topLevel *cobra.Command) {
	var sbomType string
	deps := &cobra.Command{
		Use:   "deps IMAGE",
		Short: "Print Go module dependency information about the ko-built binary in the image",
		Long: `This sub-command finds and extracts the executable binary in the image, assuming it was built by ko, and prints information about the Go module dependencies of that executable, as reported by "go version -m".

If the image was not built using ko, or if it was built without embedding dependency information, this command will fail.`,
		Example: `
  # Fetch and extract Go dependency information from an image:
  ko deps docker.io/my-user/my-image:v3`,
		Args:       cobra.ExactArgs(1),
		Deprecated: "SBOMs are generated and uploaded by default; this command will be removed in a future release.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			switch sbomType {
			case "cyclonedx", "spdx", "go.version-m":
			default:
				return fmt.Errorf("invalid sbom type %q: must be spdx, cyclonedx or go.version-m", sbomType)
			}

			ref, err := name.ParseReference(args[0])
			if err != nil {
				return err
			}

			img, err := remote.Image(ref,
				remote.WithContext(ctx),
				remote.WithAuthFromKeychain(keychain),
				remote.WithUserAgent(ua()))
			if err != nil {
				return err
			}

			cfg, err := img.ConfigFile()
			if err != nil {
				return err
			}
			ep := cfg.Config.Entrypoint
			if len(ep) != 1 {
				return fmt.Errorf("unexpected entrypoint: %s", ep)
			}
			bin := ep[0]

			rc := mutate.Extract(img)
			defer rc.Close()
			tr := tar.NewReader(rc)
			for {
				// Stop reading if the context is cancelled.
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					// keep reading.
				}
				h, err := tr.Next()
				if errors.Is(err, io.EOF) {
					return fmt.Errorf("no ko-built executable named %q found", bin)
				}
				if err != nil {
					return err
				}

				if h.Typeflag != tar.TypeReg {
					continue
				}
				if h.Name != bin {
					continue
				}

				tmp, err := ioutil.TempFile("", filepath.Base(filepath.Clean(h.Name)))
				if err != nil {
					return err
				}
				n := tmp.Name()
				defer os.RemoveAll(n) // best effort: remove tmp file afterwards.
				defer tmp.Close()     // close it first.
				// io.LimitReader to appease gosec...
				if _, err := io.Copy(tmp, io.LimitReader(tr, h.Size)); err != nil {
					return err
				}
				if err := os.Chmod(n, os.FileMode(h.Mode)); err != nil {
					return err
				}
				cmd := exec.CommandContext(ctx, "go", "version", "-m", n)
				var buf bytes.Buffer
				cmd.Stdout = &buf
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return err
				}
				// In order to get deterministics SBOMs replace
				// our randomized file name with the path the
				// app will get inside of the container.
				mod := bytes.Replace(buf.Bytes(),
					[]byte(n),
					[]byte(path.Join("/ko-app", filepath.Base(filepath.Clean(h.Name)))),
					1)
				switch sbomType {
				case "spdx":
					b, err := sbom.GenerateImageSPDX(Version, mod, signed.Image(img))
					if err != nil {
						return err
					}
					io.Copy(os.Stdout, bytes.NewReader(b))
				case "cyclonedx":
					b, err := sbom.GenerateImageCycloneDX(mod)
					if err != nil {
						return err
					}
					io.Copy(os.Stdout, bytes.NewReader(b))
				case "go.version-m":
					io.Copy(os.Stdout, bytes.NewReader(mod))
				}
				return nil
			}
			// unreachable
		},
	}
	deps.Flags().StringVar(&sbomType, "sbom", "spdx", "Format for SBOM output (supports: spdx, cyclonedx, go.version-m).")
	topLevel.AddCommand(deps)
}
