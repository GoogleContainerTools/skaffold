package writer_test

import (
	"bytes"
	"testing"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/internal/inspectimage/writer"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestTOML(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "TOML Writer", testTOML, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testTOML(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo            *client.ImageInfo
		remoteInfoNoRebasable *client.ImageInfo
		localInfo             *client.ImageInfo
		localInfoNoRebasable  *client.ImageInfo

		expectedLocalOutput = `[local_info]
stack = 'test.stack.id.local'
rebasable = true

[local_info.base_image]
top_layer = 'some-local-top-layer'
reference = 'some-local-run-image-reference'

[[local_info.run_images]]
name = 'user-configured-mirror-for-local'
user_configured = true

[[local_info.run_images]]
name = 'some-local-run-image'

[[local_info.run_images]]
name = 'some-local-mirror'

[[local_info.run_images]]
name = 'other-local-mirror'

[[local_info.buildpacks]]
id = 'test.bp.one.local'
version = '1.0.0'
homepage = 'https://some-homepage-one'

[[local_info.buildpacks]]
id = 'test.bp.two.local'
version = '2.0.0'
homepage = 'https://some-homepage-two'

[[local_info.processes]]
type = 'some-local-type'
shell = 'bash'
command = '/some/local command'
default = true
args = [
    'some',
    'local',
    'args',
]
working-dir = "/some-test-work-dir"

[[local_info.processes]]
type = 'other-local-type'
shell = ''
command = '/other/local/command'
default = false
args = [
    'other',
    'local',
    'args',
]
working-dir = "/other-test-work-dir"
`
		expectedLocalNoRebasableOutput = `[local_info]
stack = 'test.stack.id.local'
rebasable = false

[local_info.base_image]
top_layer = 'some-local-top-layer'
reference = 'some-local-run-image-reference'

[[local_info.run_images]]
name = 'user-configured-mirror-for-local'
user_configured = true

[[local_info.run_images]]
name = 'some-local-run-image'

[[local_info.run_images]]
name = 'some-local-mirror'

[[local_info.run_images]]
name = 'other-local-mirror'

[[local_info.buildpacks]]
id = 'test.bp.one.local'
version = '1.0.0'
homepage = 'https://some-homepage-one'

[[local_info.buildpacks]]
id = 'test.bp.two.local'
version = '2.0.0'
homepage = 'https://some-homepage-two'

[[local_info.processes]]
type = 'some-local-type'
shell = 'bash'
command = '/some/local command'
default = true
args = [
    'some',
    'local',
    'args',
]
working-dir = "/some-test-work-dir"

[[local_info.processes]]
type = 'other-local-type'
shell = ''
command = '/other/local/command'
default = false
args = [
    'other',
    'local',
    'args',
]
working-dir = "/other-test-work-dir"
`

		expectedRemoteOutput = `
[remote_info]
stack = 'test.stack.id.remote'
rebasable = true

[remote_info.base_image]
top_layer = 'some-remote-top-layer'
reference = 'some-remote-run-image-reference'

[[remote_info.run_images]]
name = 'user-configured-mirror-for-remote'
user_configured = true

[[remote_info.run_images]]
name = 'some-remote-run-image'

[[remote_info.run_images]]
name = 'some-remote-mirror'

[[remote_info.run_images]]
name = 'other-remote-mirror'

[[remote_info.buildpacks]]
id = 'test.bp.one.remote'
version = '1.0.0'
homepage = 'https://some-homepage-one'

[[remote_info.buildpacks]]
id = 'test.bp.two.remote'
version = '2.0.0'
homepage = 'https://some-homepage-two'

[[remote_info.processes]]
type = 'some-remote-type'
shell = 'bash'
command = '/some/remote command'
default = true
args = [
    'some',
    'remote',
    'args',
]
working-dir = "/some-test-work-dir"

[[remote_info.processes]]
type = 'other-remote-type'
shell = ''
command = '/other/remote/command'
default = false
args = [
    'other',
    'remote',
    'args',
]
working-dir = "/other-test-work-dir"
`
		expectedRemoteNoRebasableOutput = `
[remote_info]
stack = 'test.stack.id.remote'
rebasable = false

[remote_info.base_image]
top_layer = 'some-remote-top-layer'
reference = 'some-remote-run-image-reference'

[[remote_info.run_images]]
name = 'user-configured-mirror-for-remote'
user_configured = true

[[remote_info.run_images]]
name = 'some-remote-run-image'

[[remote_info.run_images]]
name = 'some-remote-mirror'

[[remote_info.run_images]]
name = 'other-remote-mirror'

[[remote_info.buildpacks]]
id = 'test.bp.one.remote'
version = '1.0.0'
homepage = 'https://some-homepage-one'

[[remote_info.buildpacks]]
id = 'test.bp.two.remote'
version = '2.0.0'
homepage = 'https://some-homepage-two'

[[remote_info.processes]]
type = 'some-remote-type'
shell = 'bash'
command = '/some/remote command'
default = true
args = [
    'some',
    'remote',
    'args',
]
working-dir = "/some-test-work-dir"

[[remote_info.processes]]
type = 'other-remote-type'
shell = ''
command = '/other/remote/command'
default = false
args = [
    'other',
    'remote',
    'args',
]
working-dir = "/other-test-work-dir"
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
				StackID: "test.stack.id.remote",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.remote", Version: "2.0.0", Homepage: "https://some-homepage-two"},
				},
				Base: files.RunImageForRebase{
					TopLayer:  "some-remote-top-layer",
					Reference: "some-remote-run-image-reference",
				},
				Stack: files.Stack{
					RunImage: files.RunImageForExport{
						Image:   "some-remote-run-image",
						Mirrors: []string{"some-remote-mirror", "other-remote-mirror"},
					},
				},
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
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-remote-type",
						Command:          launch.RawCommand{Entries: []string{"/some/remote command"}},
						Args:             []string{"some", "remote", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-remote-type",
							Command:          launch.RawCommand{Entries: []string{"/other/remote/command"}},
							Args:             []string{"other", "remote", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
				Rebasable: true,
			}
			remoteInfoNoRebasable = &client.ImageInfo{
				StackID: "test.stack.id.remote",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.remote", Version: "2.0.0", Homepage: "https://some-homepage-two"},
				},
				Base: files.RunImageForRebase{
					TopLayer:  "some-remote-top-layer",
					Reference: "some-remote-run-image-reference",
				},
				Stack: files.Stack{
					RunImage: files.RunImageForExport{
						Image:   "some-remote-run-image",
						Mirrors: []string{"some-remote-mirror", "other-remote-mirror"},
					},
				},
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
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-remote-type",
						Command:          launch.RawCommand{Entries: []string{"/some/remote command"}},
						Args:             []string{"some", "remote", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-remote-type",
							Command:          launch.RawCommand{Entries: []string{"/other/remote/command"}},
							Args:             []string{"other", "remote", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
				Rebasable: false,
			}

			localInfo = &client.ImageInfo{
				StackID: "test.stack.id.local",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.local", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.local", Version: "2.0.0", Homepage: "https://some-homepage-two"},
				},
				Base: files.RunImageForRebase{
					TopLayer:  "some-local-top-layer",
					Reference: "some-local-run-image-reference",
				},
				Stack: files.Stack{
					RunImage: files.RunImageForExport{
						Image:   "some-local-run-image",
						Mirrors: []string{"some-local-mirror", "other-local-mirror"},
					},
				},
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
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-local-type",
						Command:          launch.RawCommand{Entries: []string{"/some/local command"}},
						Args:             []string{"some", "local", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-local-type",
							Command:          launch.RawCommand{Entries: []string{"/other/local/command"}},
							Args:             []string{"other", "local", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
				Rebasable: true,
			}
			localInfoNoRebasable = &client.ImageInfo{
				StackID: "test.stack.id.local",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.local", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.local", Version: "2.0.0", Homepage: "https://some-homepage-two"},
				},
				Base: files.RunImageForRebase{
					TopLayer:  "some-local-top-layer",
					Reference: "some-local-run-image-reference",
				},
				Stack: files.Stack{
					RunImage: files.RunImageForExport{
						Image:   "some-local-run-image",
						Mirrors: []string{"some-local-mirror", "other-local-mirror"},
					},
				},
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
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-local-type",
						Command:          launch.RawCommand{Entries: []string{"/some/local command"}},
						Args:             []string{"some", "local", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-local-type",
							Command:          launch.RawCommand{Entries: []string{"/other/local/command"}},
							Args:             []string{"other", "local", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
				Rebasable: false,
			}

			outBuf = bytes.Buffer{}
		})

		when("local and remote image exits", func() {
			it("prints both local and remote image info in a TOML format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, sharedImageInfo, localInfo, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.ContainsTOML(outBuf.String(), `image_name = "test-image"`)
				assert.ContainsTOML(outBuf.String(), expectedLocalOutput)
				assert.ContainsTOML(outBuf.String(), expectedRemoteOutput)
			})
			it("prints both local and remote no rebasable images info in a TOML format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, sharedImageInfo, localInfoNoRebasable, remoteInfoNoRebasable, nil, nil)
				assert.Nil(err)

				assert.ContainsTOML(outBuf.String(), `image_name = "test-image"`)
				assert.ContainsTOML(outBuf.String(), expectedLocalNoRebasableOutput)
				assert.ContainsTOML(outBuf.String(), expectedRemoteNoRebasableOutput)
			})
		})

		when("only local image exists", func() {
			it("prints local image info in TOML format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, sharedImageInfo, localInfo, nil, nil, nil)
				assert.Nil(err)

				assert.ContainsTOML(outBuf.String(), `image_name = "test-image"`)
				assert.NotContains(outBuf.String(), "test.stack.id.remote")
				assert.ContainsTOML(outBuf.String(), expectedLocalOutput)
			})
		})

		when("only remote image exists", func() {
			it("prints remote image info in TOML format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, sharedImageInfo, nil, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.ContainsTOML(outBuf.String(), `image_name = "test-image"`)
				assert.NotContains(outBuf.String(), "test.stack.id.local")
				assert.ContainsTOML(outBuf.String(), expectedRemoteOutput)
			})
		})
	})
}
