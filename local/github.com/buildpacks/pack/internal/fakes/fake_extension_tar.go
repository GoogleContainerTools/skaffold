package fakes

import (
	"io"
	"os"
	"testing"

	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func CreateExtensionTar(t *testing.T, tmpDir string, descriptor dist.ExtensionDescriptor) string {
	extension, err := NewFakeExtensionBlob(&descriptor, 0777)
	h.AssertNil(t, err)

	tempFile, err := os.CreateTemp(tmpDir, "ex-*.tar")
	h.AssertNil(t, err)
	defer tempFile.Close()

	reader, err := extension.Open()
	h.AssertNil(t, err)

	_, err = io.Copy(tempFile, reader)
	h.AssertNil(t, err)

	return tempFile.Name()
}
