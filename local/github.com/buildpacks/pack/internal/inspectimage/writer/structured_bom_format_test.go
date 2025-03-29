package writer_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/internal/inspectimage/writer"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestStructuredBOMFormat(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "StructuredBOMFormat Writer", testStructuredBOMFormat, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testStructuredBOMFormat(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf *bytes.Buffer

		remoteInfo              *client.ImageInfo
		localInfo               *client.ImageInfo
		remoteWithExtensionInfo *client.ImageInfo
		localWithExtensionInfo  *client.ImageInfo
		generalInfo             inspectimage.GeneralInfo
		logger                  *logging.LogWithWriters
	)

	when("Print", func() {
		it.Before(func() {
			outBuf = bytes.NewBuffer(nil)
			logger = logging.NewLogWithWriters(outBuf, outBuf)
			remoteInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{
					{
						Require: buildpack.Require{
							Name:    "remote-require",
							Version: "1.2.3",
							Metadata: map[string]interface{}{
								"cool-remote": "beans",
							},
						},
						Buildpack: buildpack.GroupElement{
							ID:      "remote-buildpack",
							Version: "remote-buildpack-version",
						},
					},
				},
			}
			localInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{
					{
						Require: buildpack.Require{
							Name:    "local-require",
							Version: "4.5.6",
							Metadata: map[string]interface{}{
								"cool-local": "beans",
							},
						},
						Buildpack: buildpack.GroupElement{
							ID:      "local-buildpack",
							Version: "local-buildpack-version",
						},
					},
				},
			}

			remoteWithExtensionInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{
					{
						Require: buildpack.Require{
							Name:    "remote-require",
							Version: "1.2.3",
							Metadata: map[string]interface{}{
								"cool-remote": "beans",
							},
						},
						Buildpack: buildpack.GroupElement{
							ID:      "remote-buildpack",
							Version: "remote-buildpack-version",
						},
					},
				},
			}
			localWithExtensionInfo = &client.ImageInfo{
				BOM: []buildpack.BOMEntry{
					{
						Require: buildpack.Require{
							Name:    "local-require",
							Version: "4.5.6",
							Metadata: map[string]interface{}{
								"cool-local": "beans",
							},
						},
						Buildpack: buildpack.GroupElement{
							ID:      "local-buildpack",
							Version: "local-buildpack-version",
						},
					},
				},
			}

			generalInfo = inspectimage.GeneralInfo{
				Name: "some-image-name",
				RunImageMirrors: []config.RunImage{
					{
						Image:   "some-run-image",
						Mirrors: []string{"first-mirror", "second-mirror"},
					},
				},
			}
		})

		when("structured output", func() {
			var (
				localBomDisplay               []inspectimage.BOMEntryDisplay
				remoteBomDisplay              []inspectimage.BOMEntryDisplay
				localBomWithExtensionDisplay  []inspectimage.BOMEntryDisplay
				remoteBomWithExtensionDisplay []inspectimage.BOMEntryDisplay
			)
			it.Before(func() {
				localBomDisplay = []inspectimage.BOMEntryDisplay{{
					Name:    "local-require",
					Version: "4.5.6",
					Metadata: map[string]interface{}{
						"cool-local": "beans",
					},
					Buildpack: dist.ModuleRef{
						ModuleInfo: dist.ModuleInfo{
							ID:      "local-buildpack",
							Version: "local-buildpack-version",
						},
					},
				}}
				remoteBomDisplay = []inspectimage.BOMEntryDisplay{{
					Name:    "remote-require",
					Version: "1.2.3",
					Metadata: map[string]interface{}{
						"cool-remote": "beans",
					},
					Buildpack: dist.ModuleRef{
						ModuleInfo: dist.ModuleInfo{
							ID:      "remote-buildpack",
							Version: "remote-buildpack-version",
						},
					},
				}}

				localBomWithExtensionDisplay = []inspectimage.BOMEntryDisplay{{
					Name:    "local-require",
					Version: "4.5.6",
					Metadata: map[string]interface{}{
						"cool-local": "beans",
					},
					Buildpack: dist.ModuleRef{
						ModuleInfo: dist.ModuleInfo{
							ID:      "local-buildpack",
							Version: "local-buildpack-version",
						},
					},
				}}
				remoteBomWithExtensionDisplay = []inspectimage.BOMEntryDisplay{{
					Name:    "remote-require",
					Version: "1.2.3",
					Metadata: map[string]interface{}{
						"cool-remote": "beans",
					},
					Buildpack: dist.ModuleRef{
						ModuleInfo: dist.ModuleInfo{
							ID:      "remote-buildpack",
							Version: "remote-buildpack-version",
						},
					},
				}}
			})
			it("passes correct info to structuredBOMWriter", func() {
				var marshalInput interface{}

				structuredBOMWriter := writer.StructuredBOMFormat{
					MarshalFunc: func(i interface{}) ([]byte, error) {
						marshalInput = i
						return []byte("marshalled"), nil
					},
				}

				err := structuredBOMWriter.Print(logger, generalInfo, localInfo, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.Equal(marshalInput, inspectimage.BOMDisplay{
					Remote: remoteBomDisplay,
					Local:  localBomDisplay,
				})
			})

			it("passes correct info to structuredBOMWriter", func() {
				var marshalInput interface{}

				structuredBOMWriter := writer.StructuredBOMFormat{
					MarshalFunc: func(i interface{}) ([]byte, error) {
						marshalInput = i
						return []byte("marshalled"), nil
					},
				}

				err := structuredBOMWriter.Print(logger, generalInfo, localWithExtensionInfo, remoteWithExtensionInfo, nil, nil)
				assert.Nil(err)

				assert.Equal(marshalInput, inspectimage.BOMDisplay{
					Remote: remoteBomWithExtensionDisplay,
					Local:  localBomWithExtensionDisplay,
				})
			})
			when("a localErr is passed to Print", func() {
				it("still marshals remote information", func() {
					var marshalInput interface{}

					localErr := errors.New("a local error occurred")
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							marshalInput = i
							return []byte("marshalled"), nil
						},
					}

					err := structuredBOMWriter.Print(logger, generalInfo, nil, remoteInfo, localErr, nil)
					assert.Nil(err)

					assert.Equal(marshalInput, inspectimage.BOMDisplay{
						Remote:   remoteBomDisplay,
						Local:    nil,
						LocalErr: localErr.Error(),
					})
				})
			})

			when("a localErr is passed to Print", func() {
				it("still marshals remote information", func() {
					var marshalInput interface{}

					localErr := errors.New("a local error occurred")
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							marshalInput = i
							return []byte("marshalled"), nil
						},
					}

					err := structuredBOMWriter.Print(logger, generalInfo, nil, remoteWithExtensionInfo, localErr, nil)
					assert.Nil(err)

					assert.Equal(marshalInput, inspectimage.BOMDisplay{
						Remote:   remoteBomWithExtensionDisplay,
						Local:    nil,
						LocalErr: localErr.Error(),
					})
				})
			})

			when("a remoteErr is passed to Print", func() {
				it("still marshals local information", func() {
					var marshalInput interface{}

					remoteErr := errors.New("a remote error occurred")
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							marshalInput = i
							return []byte("marshalled"), nil
						},
					}

					err := structuredBOMWriter.Print(logger, generalInfo, localInfo, nil, nil, remoteErr)
					assert.Nil(err)

					assert.Equal(marshalInput, inspectimage.BOMDisplay{
						Remote:    nil,
						Local:     localBomDisplay,
						RemoteErr: remoteErr.Error(),
					})
				})
			})

			when("a remoteErr is passed to Print", func() {
				it("still marshals local information", func() {
					var marshalInput interface{}

					remoteErr := errors.New("a remote error occurred")
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							marshalInput = i
							return []byte("marshalled"), nil
						},
					}

					err := structuredBOMWriter.Print(logger, generalInfo, localWithExtensionInfo, nil, nil, remoteErr)
					assert.Nil(err)

					assert.Equal(marshalInput, inspectimage.BOMDisplay{
						Remote:    nil,
						Local:     localBomWithExtensionDisplay,
						RemoteErr: remoteErr.Error(),
					})
				})
			})
		})

		// Just test error cases, all error-free cases will be tested in JSON, TOML, and YAML subclasses.
		when("failure cases", func() {
			when("both info objects are nil", func() {
				it("displays a 'missing image' error message'", func() {
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: testMarshalFunc,
					}

					err := structuredBOMWriter.Print(logger, generalInfo, nil, nil, nil, nil)
					assert.ErrorWithMessage(err, fmt.Sprintf("unable to find image '%s' locally or remotely", "some-image-name"))
				})
			})
			when("fetching local and remote info errors", func() {
				it("returns an error", func() {
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							return []byte("cool"), nil
						},
					}
					remoteErr := errors.New("a remote error occurred")
					localErr := errors.New("a local error occurred")

					err := structuredBOMWriter.Print(logger, generalInfo, localInfo, remoteInfo, localErr, remoteErr)
					assert.ErrorContains(err, remoteErr.Error())
					assert.ErrorContains(err, localErr.Error())
				})
			})

			when("fetching local and remote info errors", func() {
				it("returns an error", func() {
					structuredBOMWriter := writer.StructuredBOMFormat{
						MarshalFunc: func(i interface{}) ([]byte, error) {
							return []byte("cool"), nil
						},
					}
					remoteErr := errors.New("a remote error occurred")
					localErr := errors.New("a local error occurred")

					err := structuredBOMWriter.Print(logger, generalInfo, localWithExtensionInfo, remoteWithExtensionInfo, localErr, remoteErr)
					assert.ErrorContains(err, remoteErr.Error())
					assert.ErrorContains(err, localErr.Error())
				})
			})
		})
	})
}
