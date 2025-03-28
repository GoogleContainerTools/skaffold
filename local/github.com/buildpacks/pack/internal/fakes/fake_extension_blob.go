package fakes

import (
	"bytes"
	"io"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
)

type fakeExtensionBlob struct {
	descriptor buildpack.Descriptor
	chmod      int64
}

func NewFakeExtensionBlob(descriptor buildpack.Descriptor, chmod int64) (blob.Blob, error) {
	return &fakeExtensionBlob{
		descriptor: descriptor,
		chmod:      chmod,
	}, nil
}

func (b *fakeExtensionBlob) Open() (reader io.ReadCloser, err error) {
	buf := &bytes.Buffer{}
	if err = toml.NewEncoder(buf).Encode(b.descriptor); err != nil {
		return nil, err
	}

	tarBuilder := archive.TarBuilder{}
	tarBuilder.AddFile("extension.toml", b.chmod, time.Now(), buf.Bytes())
	tarBuilder.AddDir("bin", b.chmod, time.Now())
	tarBuilder.AddFile("bin/build", b.chmod, time.Now(), []byte("build-contents"))
	tarBuilder.AddFile("bin/detect", b.chmod, time.Now(), []byte("detect-contents"))

	return tarBuilder.Reader(archive.DefaultTarWriterFactory()), err
}
