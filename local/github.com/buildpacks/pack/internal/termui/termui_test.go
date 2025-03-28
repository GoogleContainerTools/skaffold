package termui

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/rivo/tview"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/termui/fakes"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestScreen(t *testing.T) {
	spec.Run(t, "Termui", testTermui, spec.Report(report.Terminal{}))
}

func testTermui(t *testing.T, when spec.G, it spec.S) {
	var (
		assert             = h.NewAssertionManager(t)
		eventuallyInterval = 500 * time.Millisecond
		eventuallyDuration = 5 * time.Second
	)

	it("performs the lifecycle", func() {
		var (
			fakeBuild           = make(chan bool, 1)
			fakeBodyChan        = make(chan dcontainer.WaitResponse, 1)
			fakeApp             = fakes.NewApp()
			r, w                = io.Pipe()
			fakeDockerStdWriter = fakes.NewDockerStdWriter(w)

			fakeBuilder = fakes.NewBuilder("some/basename",
				[]dist.ModuleInfo{
					{ID: "some/buildpack-1", Version: "0.0.1", Homepage: "https://some/buildpack-1"},
					{ID: "some/buildpack-2", Version: "0.0.2", Homepage: "https://some/buildpack-2"},
				},
				builder.LifecycleDescriptor{Info: builder.LifecycleInfo{
					Version: builder.VersionMustParse("0.0.1"),
				}},
				builder.StackMetadata{
					RunImage: builder.RunImageMetadata{
						Image: "some/run-image",
					},
				},
			)

			s = &Termui{
				appName:       "some/app-name",
				bldr:          fakeBuilder,
				runImageName:  "some/run-image-name",
				app:           fakeApp,
				buildpackChan: make(chan dist.ModuleInfo, 10),
				textChan:      make(chan string, 10),
				nodes:         map[string]*tview.TreeNode{},
			}
		)

		defer func() {
			fakeBodyChan <- dcontainer.WaitResponse{StatusCode: 0}
			fakeBuild <- true
			w.Close()
			fakeApp.StopRunning()
		}()
		go s.Run(func() { <-fakeBuild })
		go s.Handler()(fakeBodyChan, nil, r)

		h.Eventually(t, func() bool {
			return fakeApp.SetRootCallCount == 1
		}, eventuallyInterval, eventuallyDuration)

		detectPage, ok := s.currentPage.(*Detect)
		assert.TrueWithMessage(ok, fmt.Sprintf("expected %T to be assignable to type `*screen.Detect`", s.currentPage))
		assert.TrueWithMessage(fakeApp.DrawCallCount > 0, "expect app.Draw() to be called")
		h.Eventually(t, func() bool {
			return strings.Contains(detectPage.textView.GetText(true), "Detecting")
		}, eventuallyInterval, eventuallyDuration)

		fakeDockerStdWriter.WriteStdoutln(`1 of 2 buildpacks participating`)
		fakeDockerStdWriter.WriteStdoutln(`some/buildpack-1 0.0.1`)

		// move to next screen
		fakeDockerStdWriter.WriteStdoutln(`===> BUILDING`)
		h.Eventually(t, func() bool {
			return strings.Contains(detectPage.textView.GetText(true), "Detected!")
		}, eventuallyInterval, eventuallyDuration)

		h.Eventually(t, func() bool {
			_, ok := s.currentPage.(*Dashboard)
			return ok
		}, eventuallyInterval, eventuallyDuration)
		assert.Equal(fakeApp.SetRootCallCount, 2)

		dashboardPage, ok := s.currentPage.(*Dashboard)
		assert.TrueWithMessage(ok, fmt.Sprintf("expected %T to be assignable to type `*screen.Dashboard`", s.currentPage))
		assert.Equal(dashboardPage.planList.GetItemCount(), 1)
		buildpackName, buildpackDescription := dashboardPage.planList.GetItemText(0)
		assert.Equal(buildpackName, "some/buildpack-1@0.0.1")
		assert.Equal(buildpackDescription, "https://some/buildpack-1")

		assert.Matches(dashboardPage.appTree.GetRoot().GetText(), regexp.MustCompile(`app: .*some/app-name`))
		assert.Matches(dashboardPage.appTree.GetRoot().GetChildren()[0].GetText(), regexp.MustCompile(`run: .*some/run-image-name`))
		assert.Matches(dashboardPage.builderTree.GetRoot().GetText(), regexp.MustCompile(`builder: .*some/basename`))
		assert.Matches(dashboardPage.builderTree.GetRoot().GetChildren()[0].GetText(), regexp.MustCompile(`lifecycle: .*0.0.1`))
		assert.Matches(dashboardPage.builderTree.GetRoot().GetChildren()[1].GetText(), regexp.MustCompile(`run: .*some/run-image`))
		assert.Matches(dashboardPage.builderTree.GetRoot().GetChildren()[2].GetText(), regexp.MustCompile(`buildpacks`))

		fakeDockerStdWriter.WriteStdoutln(`some-build-logs`)
		h.Eventually(t, func() bool {
			return strings.Contains(dashboardPage.logsView.GetText(true), "some-build-logs")
		}, eventuallyInterval, eventuallyDuration)

		// extract /layers from build and provide to termui
		f, err := os.Open("./testdata/fake-layers.tar")
		h.AssertNil(t, err)
		h.AssertNil(t, s.ReadLayers(f))

		bpChildren1 := dashboardPage.nodes["layers/some_buildpack-1"].GetChildren()
		h.AssertEq(t, len(bpChildren1), 1)
		h.AssertEq(t, bpChildren1[0].GetText(), "some-file-1.txt")
		h.AssertFalse(t, bpChildren1[0].GetReference().(*tar.Header).FileInfo().IsDir())

		bpChildren2 := dashboardPage.nodes["layers/some_buildpack-2"].GetChildren()
		h.AssertEq(t, len(bpChildren2), 1)
		h.AssertEq(t, bpChildren2[0].GetText(), "some-dir")
		h.AssertTrue(t, bpChildren2[0].GetReference().(*tar.Header).FileInfo().IsDir())

		h.AssertEq(t, len(bpChildren2[0].GetChildren()), 1)
		h.AssertEq(t, bpChildren2[0].GetChildren()[0].GetText(), "some-file-2.txt")
		h.AssertFalse(t, bpChildren2[0].GetChildren()[0].GetReference().(*tar.Header).FileInfo().IsDir())

		// finish build
		fakeBodyChan <- dcontainer.WaitResponse{StatusCode: 0}
		w.Close()
		time.Sleep(500 * time.Millisecond)
		fakeBuild <- true
		h.Eventually(t, func() bool {
			return strings.Contains(dashboardPage.logsView.GetText(true), "BUILD SUCCEEDED")
		}, eventuallyInterval, eventuallyDuration)
	})

	it("performs the lifecycle (when the builder is untrusted)", func() {
		var (
			fakeBuild           = make(chan bool, 1)
			fakeBodyChan        = make(chan dcontainer.WaitResponse, 1)
			fakeApp             = fakes.NewApp()
			r, w                = io.Pipe()
			fakeDockerStdWriter = fakes.NewDockerStdWriter(w)

			fakeBuilder = fakes.NewBuilder("some/basename",
				[]dist.ModuleInfo{
					{ID: "some/buildpack-1", Version: "0.0.1", Homepage: "https://some/buildpack-1"},
					{ID: "some/buildpack-2", Version: "0.0.2", Homepage: "https://some/buildpack-2"},
				},
				builder.LifecycleDescriptor{Info: builder.LifecycleInfo{
					Version: builder.VersionMustParse("0.0.1"),
				}},
				builder.StackMetadata{
					RunImage: builder.RunImageMetadata{
						Image: "some/run-image",
					},
				},
			)

			s = &Termui{
				appName:       "some/app-name",
				bldr:          fakeBuilder,
				runImageName:  "some/run-image-name",
				app:           fakeApp,
				buildpackChan: make(chan dist.ModuleInfo, 10),
				textChan:      make(chan string, 10),
			}
		)

		defer func() {
			fakeBodyChan <- dcontainer.WaitResponse{StatusCode: 0}
			fakeBuild <- true
			w.Close()
			fakeApp.StopRunning()
		}()
		go s.Run(func() { <-fakeBuild })
		go s.Handler()(fakeBodyChan, nil, r)

		h.Eventually(t, func() bool {
			return fakeApp.SetRootCallCount == 1
		}, eventuallyInterval, eventuallyDuration)

		assert.Equal(fakeApp.SetRootCallCount, 1)
		currentPage, ok := s.currentPage.(*Detect)
		assert.TrueWithMessage(ok, fmt.Sprintf("expected %T to be assignable to type `*screen.Detect`", s.currentPage))
		assert.TrueWithMessage(fakeApp.DrawCallCount > 0, "expect app.Draw() to be called")
		h.Eventually(t, func() bool {
			return strings.Contains(currentPage.textView.GetText(true), "Detecting")
		}, eventuallyInterval, eventuallyDuration)

		// move to next screen
		s.Info(`===> BUILDING`)
		h.Eventually(t, func() bool {
			return strings.Contains(currentPage.textView.GetText(true), "Detected!")
		}, eventuallyInterval, eventuallyDuration)

		h.Eventually(t, func() bool {
			_, ok := s.currentPage.(*Dashboard)
			return ok
		}, eventuallyInterval, eventuallyDuration)
		assert.Equal(fakeApp.SetRootCallCount, 2)

		dashboardPage, ok := s.currentPage.(*Dashboard)
		assert.TrueWithMessage(ok, fmt.Sprintf("expected %T to be assignable to type `*screen.Dashboard`", s.currentPage))

		fakeDockerStdWriter.WriteStdoutln(`some-build-logs`)
		h.Eventually(t, func() bool {
			return strings.Contains(dashboardPage.logsView.GetText(true), "some-build-logs")
		}, eventuallyInterval, eventuallyDuration)

		// finish build
		fakeBodyChan <- dcontainer.WaitResponse{StatusCode: 1}
		w.Close()
		time.Sleep(500 * time.Millisecond)
		fakeBuild <- true
		h.Eventually(t, func() bool {
			return strings.Contains(dashboardPage.logsView.GetText(true), "BUILD FAILED")
		}, eventuallyInterval, eventuallyDuration)
	})

	// TODO: change to show errors on-screen
	// See: https://github.com/buildpacks/pack/issues/1262
	it("returns errors from error channel", func() {
		var (
			errChan = make(chan error, 1)
			fakeApp = fakes.NewApp()
			s       = Termui{app: fakeApp}
		)

		errChan <- errors.New("some-error")

		err := s.Handler()(nil, errChan, bytes.NewReader(nil))
		assert.ErrorContains(err, "some-error")
	})
}
