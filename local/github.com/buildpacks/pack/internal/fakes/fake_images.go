package fakes

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/fakes"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

type FakeImageCreator func(name string, topLayerSha string, identifier imgutil.Identifier) *fakes.Image

func NewFakeBuilderImage(t *testing.T, tmpDir, name string, stackID, uid, gid string, metadata builder.Metadata, bpLayers dist.ModuleLayers, order dist.Order, exLayers dist.ModuleLayers, orderExtensions dist.Order, creator FakeImageCreator) *fakes.Image {
	fakeBuilderImage := creator(name, "", nil)

	h.AssertNil(t, fakeBuilderImage.SetLabel("io.buildpacks.stack.id", stackID))
	h.AssertNil(t, fakeBuilderImage.SetEnv("CNB_USER_ID", uid))
	h.AssertNil(t, fakeBuilderImage.SetEnv("CNB_GROUP_ID", gid))

	h.AssertNil(t, dist.SetLabel(fakeBuilderImage, "io.buildpacks.builder.metadata", metadata))
	h.AssertNil(t, dist.SetLabel(fakeBuilderImage, "io.buildpacks.buildpack.layers", bpLayers))

	for bpID, v := range bpLayers {
		for bpVersion, bpLayerInfo := range v {
			bpInfo := dist.ModuleInfo{
				ID:      bpID,
				Version: bpVersion,
			}

			buildpackDescriptor := dist.BuildpackDescriptor{
				WithAPI:    bpLayerInfo.API,
				WithInfo:   bpInfo,
				WithStacks: bpLayerInfo.Stacks,
				WithOrder:  bpLayerInfo.Order,
			}

			buildpackTar := CreateBuildpackTar(t, tmpDir, buildpackDescriptor)
			err := fakeBuilderImage.AddLayer(buildpackTar)
			h.AssertNil(t, err)
		}
	}

	for exID, v := range exLayers {
		for exVersion, exLayerInfo := range v {
			exInfo := dist.ModuleInfo{
				ID:      exID,
				Version: exVersion,
			}

			extensionDescriptor := dist.ExtensionDescriptor{
				WithAPI:  exLayerInfo.API,
				WithInfo: exInfo,
			}

			extensionTar := CreateExtensionTar(t, tmpDir, extensionDescriptor)
			err := fakeBuilderImage.AddLayer(extensionTar)
			h.AssertNil(t, err)
		}
	}

	h.AssertNil(t, dist.SetLabel(fakeBuilderImage, "io.buildpacks.buildpack.order", order))
	h.AssertNil(t, dist.SetLabel(fakeBuilderImage, "io.buildpacks.extension.order", orderExtensions))

	tarBuilder := archive.TarBuilder{}
	orderTomlBytes := &bytes.Buffer{}
	h.AssertNil(t, toml.NewEncoder(orderTomlBytes).Encode(orderTOML{Order: order, OrderExtensions: orderExtensions}))
	tarBuilder.AddFile("/cnb/order.toml", 0777, archive.NormalizedDateTime, orderTomlBytes.Bytes())

	orderTar := filepath.Join(tmpDir, fmt.Sprintf("order.%s.toml", h.RandString(8)))
	h.AssertNil(t, tarBuilder.WriteToPath(orderTar, archive.DefaultTarWriterFactory()))
	h.AssertNil(t, fakeBuilderImage.AddLayer(orderTar))

	return fakeBuilderImage
}

type orderTOML struct {
	Order           dist.Order `toml:"order"`
	OrderExtensions dist.Order `toml:"orderExtensions"`
}
