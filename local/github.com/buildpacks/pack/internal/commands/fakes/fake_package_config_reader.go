package fakes

import (
	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/pkg/dist"
)

type FakePackageConfigReader struct {
	ReadCalledWithArg string
	ReadReturnConfig  pubbldpkg.Config
	ReadReturnError   error

	ReadBuildpackDescriptorCalledWithArg string
	ReadBuildpackDescriptorReturn        dist.BuildpackDescriptor
	ReadExtensionDescriptorReturn        dist.ExtensionDescriptor
	ReadBuildpackDescriptorReturnError   error
}

func (r *FakePackageConfigReader) Read(path string) (pubbldpkg.Config, error) {
	r.ReadCalledWithArg = path

	return r.ReadReturnConfig, r.ReadReturnError
}

func (r *FakePackageConfigReader) ReadBuildpackDescriptor(path string) (dist.BuildpackDescriptor, error) {
	r.ReadBuildpackDescriptorCalledWithArg = path

	return r.ReadBuildpackDescriptorReturn, r.ReadBuildpackDescriptorReturnError
}

func NewFakePackageConfigReader(ops ...func(*FakePackageConfigReader)) *FakePackageConfigReader {
	fakePackageConfigReader := &FakePackageConfigReader{
		ReadReturnConfig: pubbldpkg.Config{},
		ReadReturnError:  nil,
	}

	for _, op := range ops {
		op(fakePackageConfigReader)
	}

	return fakePackageConfigReader
}
