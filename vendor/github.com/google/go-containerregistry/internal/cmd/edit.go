// Copyright 2022 Google LLC All Rights Reserved.
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
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/internal/editor"
	"github.com/google/go-containerregistry/internal/verify"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"
)

// NewCmdEdit creates a new cobra.Command for the edit subcommand.
//
// This is currently hidden until we're happy with the interface and can test
// it on different operating systems and editors.
func NewCmdEdit(options *[]crane.Option) *cobra.Command {
	cmd := &cobra.Command{
		Hidden: true,
		Use:    "edit",
		Short:  "Edit the contents of an image.",
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Usage()
		},
	}
	cmd.AddCommand(NewCmdEditManifest(options), NewCmdEditConfig(options), NewCmdEditFs(options))

	return cmd
}

// NewCmdConfig creates a new cobra.Command for the config subcommand.
func NewCmdEditConfig(options *[]crane.Option) *cobra.Command {
	var dst string
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Edit an image's config file.",
		Example: `  # Edit ubuntu's config file
  crane edit config ubuntu

  # Overwrite ubuntu's config file with '{}'
  echo '{}' | crane edit config ubuntu`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := editConfig(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], dst, *options...)
			if err != nil {
				return fmt.Errorf("editing config: %w", err)
			}
			fmt.Println(ref.String())
			return nil
		},
	}
	cmd.Flags().StringVarP(&dst, "tag", "t", "", "New tag reference to apply to mutated image. If not provided, uses original tag or pushes a new digest.")

	return cmd
}

// NewCmdManifest creates a new cobra.Command for the manifest subcommand.
func NewCmdEditManifest(options *[]crane.Option) *cobra.Command {
	var (
		dst string
		mt  string
	)
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Edit an image's manifest.",
		Example: `  # Edit ubuntu's manifest
  crane edit manifest ubuntu`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := editManifest(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], dst, mt, *options...)
			if err != nil {
				return fmt.Errorf("editing manifest: %w", err)
			}
			fmt.Println(ref.String())
			return nil
		},
	}
	cmd.Flags().StringVarP(&dst, "tag", "t", "", "New tag reference to apply to mutated image. If not provided, uses original tag or pushes a new digest.")
	cmd.Flags().StringVarP(&mt, "media-type", "m", "", "Override the mediaType used as the Content-Type for PUT")

	return cmd
}

// NewCmdExport creates a new cobra.Command for the export subcommand.
func NewCmdEditFs(options *[]crane.Option) *cobra.Command {
	var dst, name string
	cmd := &cobra.Command{
		Use:   "fs IMAGE",
		Short: "Edit the contents of an image's filesystem.",
		Example: `  # Edit motd-news using $EDITOR
  crane edit fs ubuntu -f /etc/default/motd-news

  # Overwrite motd-news with 'ENABLED=0'
  echo 'ENABLED=0' | crane edit fs ubuntu -f /etc/default/motd-news`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := editFile(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], name, dst, *options...)
			if err != nil {
				return fmt.Errorf("editing file: %w", err)
			}
			fmt.Println(ref.String())
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "filename", "f", "", "Edit the given filename")
	cmd.Flags().StringVarP(&dst, "tag", "t", "", "New tag reference to apply to mutated image. If not provided, uses original tag or pushes a new digest.")
	cmd.MarkFlagRequired("filename")

	return cmd
}

func interactive(in io.Reader, out io.Writer) bool {
	return interactiveFile(in) && interactiveFile(out)
}

func interactiveFile(i any) bool {
	f, ok := i.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func editConfig(in io.Reader, out io.Writer, src, dst string, options ...crane.Option) (name.Reference, error) {
	o := crane.GetOptions(options...)

	img, err := crane.Pull(src, options...)
	if err != nil {
		return nil, err
	}

	m, err := img.Manifest()
	if err != nil {
		return nil, err
	}
	mt, err := img.MediaType()
	if err != nil {
		return nil, err
	}

	var edited []byte
	if interactive(in, out) {
		rcf, err := img.RawConfigFile()
		if err != nil {
			return nil, err
		}
		edited, err = editor.Edit(bytes.NewReader(rcf), ".json")
		if err != nil {
			return nil, err
		}
	} else {
		b, err := io.ReadAll(in)
		if err != nil {
			return nil, err
		}
		edited = b
	}

	// this has to happen before we modify the descriptor (so we can use verify.Descriptor to validate whether m.Config.Data matches m.Config.Digest/Size)
	if m.Config.Data != nil && verify.Descriptor(m.Config) == nil {
		// https://github.com/google/go-containerregistry/issues/1552#issuecomment-1452653875
		// "if data is non-empty and correct, we should update it"
		m.Config.Data = edited
	}

	l := static.NewLayer(edited, m.Config.MediaType)
	layerDigest, err := l.Digest()
	if err != nil {
		return nil, err
	}

	m.Config.Digest = layerDigest
	m.Config.Size = int64(len(edited))
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	rm := &rawManifest{
		body:      b,
		mediaType: mt,
	}

	digest, _, _ := v1.SHA256(bytes.NewReader(b))

	if dst == "" {
		dst = src
		ref, err := name.ParseReference(src, o.Name...)
		if err != nil {
			return nil, err
		}
		if _, ok := ref.(name.Digest); ok {
			dst = ref.Context().Digest(digest.String()).String()
		}
	}

	dstRef, err := name.ParseReference(dst, o.Name...)
	if err != nil {
		return nil, err
	}

	if err := remote.WriteLayer(dstRef.Context(), l, o.Remote...); err != nil {
		return nil, err
	}

	if err := remote.Put(dstRef, rm, o.Remote...); err != nil {
		return nil, err
	}

	return dstRef, nil
}

func editManifest(in io.Reader, out io.Writer, src string, dst string, mt string, options ...crane.Option) (name.Reference, error) {
	o := crane.GetOptions(options...)

	ref, err := name.ParseReference(src, o.Name...)
	if err != nil {
		return nil, err
	}

	desc, err := remote.Get(ref, o.Remote...)
	if err != nil {
		return nil, err
	}

	var edited []byte
	if interactive(in, out) {
		edited, err = editor.Edit(bytes.NewReader(desc.Manifest), ".json")
		if err != nil {
			return nil, err
		}
	} else {
		b, err := io.ReadAll(in)
		if err != nil {
			return nil, err
		}
		edited = b
	}

	digest, _, err := v1.SHA256(bytes.NewReader(edited))
	if err != nil {
		return nil, err
	}

	if dst == "" {
		dst = src
		if _, ok := ref.(name.Digest); ok {
			dst = ref.Context().Digest(digest.String()).String()
		}
	}
	dstRef, err := name.ParseReference(dst, o.Name...)
	if err != nil {
		return nil, err
	}

	if mt == "" {
		// If --media-type is unset, use Content-Type by default.
		mt = string(desc.MediaType)

		// If document contains mediaType, default to that.
		wmt := withMediaType{}
		if err := json.Unmarshal(edited, &wmt); err == nil {
			if wmt.MediaType != "" {
				mt = wmt.MediaType
			}
		}
	}

	rm := &rawManifest{
		body:      edited,
		mediaType: types.MediaType(mt),
	}

	if err := remote.Put(dstRef, rm, o.Remote...); err != nil {
		return nil, err
	}

	return dstRef, nil
}

func editFile(in io.Reader, out io.Writer, src, file, dst string, options ...crane.Option) (name.Reference, error) {
	o := crane.GetOptions(options...)

	img, err := crane.Pull(src, options...)
	if err != nil {
		return nil, err
	}

	// If stdin has content, read it in and use that for the file.
	// Otherwise, scran through the image and open that file in an editor.
	var (
		edited []byte
		header *tar.Header
	)
	if interactive(in, out) {
		f, h, err := findFile(img, file)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(h.Name)
		if strings.Contains(ext, "..") {
			return nil, fmt.Errorf("this is impossible but this check satisfies CWE-22 for file name %q", h.Name)
		}
		edited, err = editor.Edit(f, ext)
		if err != nil {
			return nil, err
		}
		header = h
	} else {
		b, err := io.ReadAll(in)
		if err != nil {
			return nil, err
		}
		edited = b
		header = blankHeader(file)
	}

	buf := bytes.NewBuffer(nil)
	buf.Grow(len(edited))
	tw := tar.NewWriter(buf)

	header.Size = int64(len(edited))
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := io.Copy(tw, bytes.NewReader(edited)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	fileBytes := buf.Bytes()
	fileLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(fileBytes)), nil
	})
	if err != nil {
		return nil, err
	}
	img, err = mutate.Append(img, mutate.Addendum{
		Layer: fileLayer,
		History: v1.History{
			Author:    "crane",
			CreatedBy: strings.Join(os.Args, " "),
		},
	})
	if err != nil {
		return nil, err
	}

	digest, err := img.Digest()
	if err != nil {
		return nil, err
	}

	if dst == "" {
		dst = src
		ref, err := name.ParseReference(src, o.Name...)
		if err != nil {
			return nil, err
		}
		if _, ok := ref.(name.Digest); ok {
			dst = ref.Context().Digest(digest.String()).String()
		}
	}

	dstRef, err := name.ParseReference(dst, o.Name...)
	if err != nil {
		return nil, err
	}

	if err := crane.Push(img, dst, options...); err != nil {
		return nil, err
	}

	return dstRef, nil
}

func findFile(img v1.Image, name string) (io.Reader, *tar.Header, error) {
	name = normalize(name)
	tr := tar.NewReader(mutate.Extract(img))
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading tar: %w", err)
		}
		if normalize(header.Name) == name {
			return tr, header, nil
		}
	}

	// If we don't find the file, we should create a new one.
	return bytes.NewBufferString(""), blankHeader(name), nil
}

func blankHeader(name string) *tar.Header {
	return &tar.Header{
		Name:     name,
		Typeflag: tar.TypeReg,
		// Use a fixed Mode, so that this isn't sensitive to the directory and umask
		// under which it was created. Additionally, windows can only set 0222,
		// 0444, or 0666, none of which are executable.
		Mode: 0555,
	}
}

func normalize(name string) string {
	return filepath.Clean("/" + name)
}

type withMediaType struct {
	MediaType string `json:"mediaType,omitempty"`
}

type rawManifest struct {
	body      []byte
	mediaType types.MediaType
}

func (r *rawManifest) RawManifest() ([]byte, error) {
	return r.body, nil
}

func (r *rawManifest) MediaType() (types.MediaType, error) {
	return r.mediaType, nil
}
