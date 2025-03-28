package writer_test

import (
	"bytes"
	"testing"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/internal/inspectimage/writer"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestYAMLBOM(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "YAML BOM Writer", testYAMLBOM, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testYAMLBOM(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo *client.ImageInfo
		localInfo  *client.ImageInfo

		expectedLocalOutput = `---
local:
- name: name-1
  version: version-1
  metadata:
    LocalData:
      string: ''
      bool: false
      int: 456
      nested:
        string: ''
  buildpacks:
    id: test.bp.one.remote
    version: 1.0.0
`
		expectedRemoteOutput = `---
remote:
- name: name-1
  version: version-1
  metadata:
    RemoteData:
      string: aString
      bool: true
      int: 123
      nested:
        string: anotherString
  buildpacks:
    id: test.bp.one.remote
    version: 1.0.0
`
	)

	when("Print", func() {
		it.Before(func() {
			type someData struct {
				String string
				Bool   bool
				Int    int
				Nested struct {
					String string
				}
			}

			remoteInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{{
					Require: buildpack.Require{
						Name:    "name-1",
						Version: "version-1",
						Metadata: map[string]interface{}{
							"RemoteData": someData{
								String: "aString",
								Bool:   true,
								Int:    123,
								Nested: struct {
									String string
								}{
									String: "anotherString",
								},
							},
						},
					},
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage"},
				}}}

			localInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{{
					Require: buildpack.Require{
						Name:    "name-1",
						Version: "version-1",
						Metadata: map[string]interface{}{
							"LocalData": someData{
								Bool: false,
								Int:  456,
							},
						},
					},
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage"},
				}},
			}

			outBuf = bytes.Buffer{}
		})

		when("local and remote image exits", func() {
			it("prints both local and remote image info in a YAML format", func() {
				yamlBOMWriter := writer.NewYAMLBOM()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlBOMWriter.Print(logger, inspectimage.GeneralInfo{}, localInfo, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.ContainsYAML(outBuf.String(), expectedLocalOutput)
				assert.ContainsYAML(outBuf.String(), expectedRemoteOutput)
			})
		})

		when("only local image exists", func() {
			it("prints local image info in YAML format", func() {
				yamlBOMWriter := writer.NewYAMLBOM()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlBOMWriter.Print(logger, inspectimage.GeneralInfo{}, localInfo, nil, nil, nil)
				assert.Nil(err)

				assert.ContainsYAML(outBuf.String(), expectedLocalOutput)

				assert.NotContains(outBuf.String(), "test.stack.id.remote")
				assert.ContainsYAML(outBuf.String(), expectedLocalOutput)
			})
		})

		when("only remote image exists", func() {
			it("prints remote image info in YAML format", func() {
				yamlBOMWriter := writer.NewYAMLBOM()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlBOMWriter.Print(logger, inspectimage.GeneralInfo{}, nil, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.NotContains(outBuf.String(), "test.stack.id.local")
				assert.ContainsYAML(outBuf.String(), expectedRemoteOutput)
			})
		})
	})
}
