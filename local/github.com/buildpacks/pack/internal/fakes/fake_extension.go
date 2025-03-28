package fakes

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

type fakeExtension struct {
	descriptor dist.ExtensionDescriptor
	chmod      int64
	options    []FakeExtensionOption
}

type fakeExtensionConfig struct {
	// maping of extrafilename to stringified contents
	ExtraFiles map[string]string
	OpenError  error
}

func newFakeExtensionConfig() *fakeExtensionConfig {
	return &fakeExtensionConfig{ExtraFiles: map[string]string{}}
}

type FakeExtensionOption func(*fakeExtensionConfig)

func WithExtraExtensionContents(filename, contents string) FakeExtensionOption {
	return func(f *fakeExtensionConfig) {
		f.ExtraFiles[filename] = contents
	}
}

func WithExtOpenError(err error) FakeExtensionOption {
	return func(f *fakeExtensionConfig) {
		f.OpenError = err
	}
}

// NewFakeExtension creates a fake extension with contents:
//
//		\_ /cnb/extensions/{ID}
//		\_ /cnb/extensions/{ID}/{version}
//		\_ /cnb/extensions/{ID}/{version}/extension.toml
//		\_ /cnb/extensions/{ID}/{version}/bin
//		\_ /cnb/extensions/{ID}/{version}/bin/generate
//	 	generate-contents
//		\_ /cnb/extensions/{ID}/{version}/bin/detect
//	 	detect-contents
func NewFakeExtension(descriptor dist.ExtensionDescriptor, chmod int64, options ...FakeExtensionOption) (buildpack.BuildModule, error) {
	return &fakeExtension{
		descriptor: descriptor,
		chmod:      chmod,
		options:    options,
	}, nil
}

func (b *fakeExtension) Descriptor() buildpack.Descriptor {
	return &b.descriptor
}

func (b *fakeExtension) Open() (io.ReadCloser, error) {
	fConfig := newFakeExtensionConfig()
	for _, option := range b.options {
		option(fConfig)
	}

	if fConfig.OpenError != nil {
		return nil, fConfig.OpenError
	}

	buf := &bytes.Buffer{}
	if err := toml.NewEncoder(buf).Encode(b.descriptor); err != nil {
		return nil, err
	}

	tarBuilder := archive.TarBuilder{}
	ts := archive.NormalizedDateTime
	tarBuilder.AddDir(fmt.Sprintf("/cnb/extensions/%s", b.descriptor.EscapedID()), b.chmod, ts)
	extDir := fmt.Sprintf("/cnb/extensions/%s/%s", b.descriptor.EscapedID(), b.descriptor.Info().Version)
	tarBuilder.AddDir(extDir, b.chmod, ts)
	tarBuilder.AddFile(extDir+"/extension.toml", b.chmod, ts, buf.Bytes())

	tarBuilder.AddDir(extDir+"/bin", b.chmod, ts)
	tarBuilder.AddFile(extDir+"/bin/generate", b.chmod, ts, []byte("generate-contents"))
	tarBuilder.AddFile(extDir+"/bin/detect", b.chmod, ts, []byte("detect-contents"))

	for extraFilename, extraContents := range fConfig.ExtraFiles {
		tarBuilder.AddFile(filepath.Join(extDir, extraFilename), b.chmod, ts, []byte(extraContents))
	}

	return tarBuilder.Reader(archive.DefaultTarWriterFactory()), nil
}
