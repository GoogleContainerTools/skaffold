package writer_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/lifecycle/buildpack"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/internal/inspectimage/writer"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestStructuredFormat(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "StructuredFormat Writer", testStructuredFormat, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testStructuredFormat(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo                    *client.ImageInfo
		localInfo                     *client.ImageInfo
		remoteWithExtensionInfo       *client.ImageInfo
		localWithExtensionInfo        *client.ImageInfo
		localInfoWithExtensionDisplay *inspectimage.InfoDisplay
	)

	when("Print", func() {
		it.Before(func() {
			remoteInfo = &client.ImageInfo{}
			localInfo = &client.ImageInfo{}
			remoteWithExtensionInfo = &client.ImageInfo{}
			localWithExtensionInfo = &client.ImageInfo{
				StackID: "test.stack.id.local",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.local", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				},
				Extensions: []buildpack.GroupElement{
					{ID: "test.bp.one.local", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				},
			}
			localInfoWithExtensionDisplay = &inspectimage.InfoDisplay{
				StackID: "test.stack.id.local",
				Buildpacks: []dist.ModuleInfo{
					{
						ID:       "test.bp.one.local",
						Version:  "1.0.0",
						Homepage: "https://some-homepage-one",
					},
				},
				Extensions: []dist.ModuleInfo{
					{
						ID:       "test.bp.one.local",
						Version:  "1.0.0",
						Homepage: "https://some-homepage-one",
					},
				},
			}
			outBuf = bytes.Buffer{}
		})

		// Just test error cases, all error-free cases will be tested in JSON, TOML, and YAML subclasses.
		when("failure cases", func() {
			when("both info objects are nil", func() {
				it("displays a 'missing image' error message'", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "missing-image",
						RunImageMirrors: []config.RunImage{},
					}

					structuredWriter := writer.StructuredFormat{
						MarshalFunc: testMarshalFunc,
					}

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, nil, nil, nil, nil)
					assert.ErrorWithMessage(err, fmt.Sprintf("unable to find image '%s' locally or remotely", "missing-image"))
				})
			})
			when("a localErr is passed to Print", func() {
				it("still prints remote information", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "localErr-image",
						RunImageMirrors: []config.RunImage{},
					}
					structuredWriter := writer.StructuredFormat{
						MarshalFunc: testMarshalFunc,
					}

					localErr := errors.New("a local error occurred")

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, nil, remoteInfo, localErr, nil)
					assert.ErrorWithMessage(err, "preparing output for 'localErr-image': a local error occurred")
				})
			})

			when("a localWithExtension is passed to Print", func() {
				it("prints localWithExtension information", func() {
					var marshalInput interface{}
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "localExtension-image",
						RunImageMirrors: []config.RunImage{},
					}
					structuredWriter := writer.StructuredFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							marshalInput = i
							return []byte("marshalled"), nil
						},
					}

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, localWithExtensionInfo, nil, nil, nil)
					assert.Nil(err)
					assert.Equal(marshalInput, inspectimage.InspectOutput{
						ImageName: "localExtension-image",
						Local:     localInfoWithExtensionDisplay,
					})
				})
			})

			when("a localErr is passed to Print", func() {
				it("still prints remote information", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "localErr-image",
						RunImageMirrors: []config.RunImage{},
					}
					structuredWriter := writer.StructuredFormat{
						MarshalFunc: testMarshalFunc,
					}

					localErr := errors.New("a local error occurred")

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, nil, remoteWithExtensionInfo, localErr, nil)
					assert.ErrorWithMessage(err, "preparing output for 'localErr-image': a local error occurred")
				})
			})

			when("a remoteErr is passed to print", func() {
				it("still prints local information", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "remoteErr-image",
						RunImageMirrors: []config.RunImage{},
					}
					structuredWriter := writer.StructuredFormat{
						MarshalFunc: testMarshalFunc,
					}

					remoteErr := errors.New("a remote error occurred")

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, localInfo, nil, nil, remoteErr)
					assert.ErrorWithMessage(err, "preparing output for 'remoteErr-image': a remote error occurred")
				})
			})

			when("a remoteErr is passed to print", func() {
				it("still prints local information", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "remoteErr-image",
						RunImageMirrors: []config.RunImage{},
					}
					structuredWriter := writer.StructuredFormat{
						MarshalFunc: testMarshalFunc,
					}

					remoteErr := errors.New("a remote error occurred")

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := structuredWriter.Print(logger, sharedImageInfo, localWithExtensionInfo, nil, nil, remoteErr)
					assert.ErrorWithMessage(err, "preparing output for 'remoteErr-image': a remote error occurred")
				})
			})
		})
	})
}

//
// testfunctions and helpers
//

func testMarshalFunc(i interface{}) ([]byte, error) {
	return []byte("marshalled"), nil
}
