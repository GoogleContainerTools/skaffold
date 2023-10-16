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

package cmd

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/spf13/cobra"
)

// NewCmdFlatten creates a new cobra.Command for the flatten subcommand.
func NewCmdFlatten(options *[]crane.Option) *cobra.Command {
	var dst string

	flattenCmd := &cobra.Command{
		Use:   "flatten",
		Short: "Flatten an image's layers into a single layer",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// We need direct access to the underlying remote options because crane
			// doesn't expose great facilities for working with an index (yet).
			o := crane.GetOptions(*options...)

			// Pull image and get config.
			src := args[0]

			// If the new ref isn't provided, write over the original image.
			// If that ref was provided by digest (e.g., output from
			// another crane command), then strip that and push the
			// mutated image by digest instead.
			if dst == "" {
				dst = src
			}

			ref, err := name.ParseReference(src, o.Name...)
			if err != nil {
				log.Fatalf("parsing %s: %v", src, err)
			}
			newRef, err := name.ParseReference(dst, o.Name...)
			if err != nil {
				log.Fatalf("parsing %s: %v", dst, err)
			}
			repo := newRef.Context()

			flat, err := flatten(ref, repo, cmd.Parent().Use, o)
			if err != nil {
				log.Fatalf("flattening %s: %v", ref, err)
			}

			digest, err := flat.Digest()
			if err != nil {
				log.Fatalf("digesting new image: %v", err)
			}

			if _, ok := ref.(name.Digest); ok {
				newRef = repo.Digest(digest.String())
			}

			if err := push(flat, newRef, o); err != nil {
				log.Fatalf("pushing %s: %v", newRef, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), repo.Digest(digest.String()))
		},
	}
	flattenCmd.Flags().StringVarP(&dst, "tag", "t", "", "New tag to apply to flattened image. If not provided, push by digest to the original image repository.")
	return flattenCmd
}

func flatten(ref name.Reference, repo name.Repository, use string, o crane.Options) (partial.Describable, error) {
	desc, err := remote.Get(ref, o.Remote...)
	if err != nil {
		return nil, fmt.Errorf("pulling %s: %w", ref, err)
	}

	if desc.MediaType.IsIndex() {
		idx, err := desc.ImageIndex()
		if err != nil {
			return nil, err
		}
		return flattenIndex(idx, repo, use, o)
	} else if desc.MediaType.IsImage() {
		img, err := desc.Image()
		if err != nil {
			return nil, err
		}
		return flattenImage(img, repo, use, o)
	}

	return nil, fmt.Errorf("can't flatten %s", desc.MediaType)
}

func push(flat partial.Describable, ref name.Reference, o crane.Options) error {
	if idx, ok := flat.(v1.ImageIndex); ok {
		return remote.WriteIndex(ref, idx, o.Remote...)
	} else if img, ok := flat.(v1.Image); ok {
		return remote.Write(ref, img, o.Remote...)
	}

	return fmt.Errorf("can't push %T", flat)
}

func flattenIndex(old v1.ImageIndex, repo name.Repository, use string, o crane.Options) (partial.Describable, error) {
	m, err := old.IndexManifest()
	if err != nil {
		return nil, err
	}

	manifests, err := partial.Manifests(old)
	if err != nil {
		return nil, err
	}

	adds := []mutate.IndexAddendum{}

	for _, m := range manifests {
		// Keep the old descriptor (annotations and whatnot).
		desc, err := partial.Descriptor(m)
		if err != nil {
			return nil, err
		}

		// Drop attestations (for now).
		// https://github.com/google/go-containerregistry/issues/1622
		if p := desc.Platform; p != nil {
			if p.OS == "unknown" && p.Architecture == "unknown" {
				continue
			}
		}

		flattened, err := flattenChild(m, repo, use, o)
		if err != nil {
			return nil, err
		}
		desc.Size, err = flattened.Size()
		if err != nil {
			return nil, err
		}
		desc.Digest, err = flattened.Digest()
		if err != nil {
			return nil, err
		}
		adds = append(adds, mutate.IndexAddendum{
			Add:        flattened,
			Descriptor: *desc,
		})
	}

	idx := mutate.AppendManifests(empty.Index, adds...)

	// Retain any annotations from the original index.
	if len(m.Annotations) != 0 {
		idx = mutate.Annotations(idx, m.Annotations).(v1.ImageIndex)
	}

	// This is stupid, but some registries get mad if you try to push OCI media types that reference docker media types.
	mt, err := old.MediaType()
	if err != nil {
		return nil, err
	}
	idx = mutate.IndexMediaType(idx, mt)

	return idx, nil
}

func flattenChild(old partial.Describable, repo name.Repository, use string, o crane.Options) (partial.Describable, error) {
	if idx, ok := old.(v1.ImageIndex); ok {
		return flattenIndex(idx, repo, use, o)
	} else if img, ok := old.(v1.Image); ok {
		return flattenImage(img, repo, use, o)
	}

	logs.Warn.Printf("can't flatten %T, skipping", old)
	return old, nil
}

func flattenImage(old v1.Image, repo name.Repository, use string, o crane.Options) (partial.Describable, error) {
	digest, err := old.Digest()
	if err != nil {
		return nil, fmt.Errorf("getting old digest: %w", err)
	}
	m, err := old.Manifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	cf, err := old.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}
	cf = cf.DeepCopy()

	oldHistory, err := json.Marshal(cf.History)
	if err != nil {
		return nil, fmt.Errorf("marshal history")
	}

	// Clear layer-specific config file information.
	cf.RootFS.DiffIDs = []v1.Hash{}
	cf.History = []v1.History{}

	img, err := mutate.ConfigFile(empty.Image, cf)
	if err != nil {
		return nil, fmt.Errorf("mutating config: %w", err)
	}

	// TODO: Make compression configurable?
	layer := stream.NewLayer(mutate.Extract(old), stream.WithCompressionLevel(gzip.BestCompression))
	if err := remote.WriteLayer(repo, layer, o.Remote...); err != nil {
		return nil, fmt.Errorf("uploading layer: %w", err)
	}

	img, err = mutate.Append(img, mutate.Addendum{
		Layer: layer,
		History: v1.History{
			CreatedBy: fmt.Sprintf("%s flatten %s", use, digest),
			Comment:   string(oldHistory),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("appending layers: %w", err)
	}

	// Retain any annotations from the original image.
	if len(m.Annotations) != 0 {
		img = mutate.Annotations(img, m.Annotations).(v1.Image)
	}

	return img, nil
}
