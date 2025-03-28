package testhelpers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buildpacks/imgutil/fakes"

	dockertypes "github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/heroku/color"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}

// Assert deep equality (and provide useful difference as a test failure)
func AssertEq(t *testing.T, actual, expected interface{}, opts ...cmp.Option) {
	t.Helper()
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Fatal(diff)
	}
}

func AssertFunctionName(t *testing.T, fn interface{}, expected string) {
	t.Helper()
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if name == "" {
		t.Fatalf("Unable to retrieve function name for %#v. Is it a function?", fn)
	}

	if !hasMatches(name, fmt.Sprintf(`\.(%s)\.func[\d]+$`, expected)) {
		t.Fatalf("Expected func name '%s' to contain '%s'", name, expected)
	}
}

// Assert deep equality (and provide useful difference as a test failure)
func AssertNotEq(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if diff := cmp.Diff(expected, actual); diff == "" {
		t.Fatal(diff)
	}
}

func AssertTrue(t *testing.T, actual interface{}) {
	t.Helper()
	AssertEq(t, actual, true)
}

func AssertFalse(t *testing.T, actual interface{}) {
	t.Helper()
	AssertEq(t, actual, false)
}

func AssertUnique(t *testing.T, items ...interface{}) {
	t.Helper()
	itemMap := map[interface{}]interface{}{}
	for _, item := range items {
		itemMap[item] = nil
	}
	if len(itemMap) != len(items) {
		t.Fatalf("Expected items in %v to be unique", items)
	}
}

// Assert the simplistic pointer (or literal value) equality
func AssertSameInstance(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if actual != expected {
		t.Fatalf("Expected %s and %s to be the same instance", actual, expected)
	}
}

func AssertError(t *testing.T, actual error, expected string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("Expected an error but got nil")
	}
	if !strings.Contains(actual.Error(), expected) {
		t.Fatalf(`Expected error to contain "%s", got "%s"`, expected, actual.Error())
	}
}

func AssertContains(t *testing.T, actual, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Fatalf(
			"Expected '%s' to contain '%s'\n\nDiff:%s",
			actual,
			expected,
			cmp.Diff(expected, actual),
		)
	}
}

func AssertContainsAllInOrder(t *testing.T, actual bytes.Buffer, expected ...string) {
	t.Helper()

	var tested []byte

	for _, exp := range expected {
		b, found := readUntilString(&actual, exp)
		tested = append(tested, b...)

		if !found {
			t.Fatalf("Expected '%s' to include all of '%s' in order", string(tested), strings.Join(expected, ", "))
		}
	}
}

func readUntilString(b *bytes.Buffer, expected string) (read []byte, found bool) {
	for {
		s, err := b.ReadBytes(expected[len(expected)-1])
		if err != nil {
			return append(read, s...), false
		}

		read = append(read, s...)
		if bytes.HasSuffix(read, []byte(expected)) {
			return read, true
		}
	}
}

// AssertContainsMatch matches on content by regular expression
func AssertContainsMatch(t *testing.T, actual, exp string) {
	t.Helper()
	if !hasMatches(actual, exp) {
		t.Fatalf("Expected '%s' to match expression '%s'", actual, exp)
	}
}

func AssertNotContainsMatch(t *testing.T, actual, exp string) {
	t.Helper()
	if hasMatches(actual, exp) {
		t.Fatalf("Expected '%s' not to match expression '%s'", actual, exp)
	}
}

func AssertNotContains(t *testing.T, actual, expected string) {
	t.Helper()
	if strings.Contains(actual, expected) {
		t.Fatalf("Expected '%s' to not contain '%s'", actual, expected)
	}
}

type KeyValue[k comparable, v any] struct {
	key   k
	value v
}

func NewKeyValue[k comparable, v any](key k, value v) KeyValue[k, v] {
	return KeyValue[k, v]{key: key, value: value}
}

func AssertMapContains[key comparable, value any](t *testing.T, actual map[key]value, expected ...KeyValue[key, value]) {
	t.Helper()
	for _, i := range expected {
		if v, ok := actual[i.key]; !ok || !reflect.DeepEqual(v, i.value) {
			t.Fatalf("Expected %s to contain elements %s", reflect.ValueOf(actual), reflect.ValueOf(expected))
		}
	}
}

func AssertMapNotContains[key comparable, value any](t *testing.T, actual map[key]value, expected ...KeyValue[key, value]) {
	t.Helper()
	for _, i := range expected {
		if v, ok := actual[i.key]; ok && reflect.DeepEqual(v, i.value) {
			t.Fatalf("Expected %s to not contain elements %s", reflect.ValueOf(actual), reflect.ValueOf(expected))
		}
	}
}

func AssertSliceContains(t *testing.T, slice []string, expected ...string) {
	t.Helper()
	_, missing, _ := stringset.Compare(slice, expected)
	if len(missing) > 0 {
		t.Fatalf("Expected %s to contain elements %s", slice, missing)
	}
}

func AssertSliceContainsInOrder(t *testing.T, slice []string, expected ...string) {
	t.Helper()

	AssertSliceContains(t, slice, expected...)

	var common []string
	expectedSet := stringset.FromSlice(expected)
	for _, sliceV := range slice {
		if _, ok := expectedSet[sliceV]; ok {
			common = append(common, sliceV)
		}
	}

	lastFoundI := -1
	for _, expectedV := range expected {
		for foundI, foundV := range common {
			if expectedV == foundV && lastFoundI < foundI {
				lastFoundI = foundI
			} else if expectedV == foundV {
				t.Fatalf("Expected '%s' come earlier in the slice.\nslice: %v\nexpected order: %v", expectedV, slice, expected)
			}
		}
	}
}

func AssertSliceNotContains(t *testing.T, slice []string, expected ...string) {
	t.Helper()
	_, missing, _ := stringset.Compare(slice, expected)
	if len(missing) != len(expected) {
		t.Fatalf("Expected %s not to contain elements %s", slice, expected)
	}
}

func AssertSliceContainsMatch(t *testing.T, slice []string, expected ...string) {
	t.Helper()

	var missing []string

	for _, expectedStr := range expected {
		var found bool
		for _, actualStr := range slice {
			if regexp.MustCompile(expectedStr).MatchString(actualStr) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, expectedStr)
		}
	}

	if len(missing) > 0 {
		t.Fatalf("Expected %s to contain elements %s", slice, missing)
	}
}

func AssertSliceContainsOnly(t *testing.T, slice []string, expected ...string) {
	t.Helper()
	extra, missing, _ := stringset.Compare(slice, expected)
	if len(missing) > 0 {
		t.Fatalf("Expected %s to contain elements %s", slice, missing)
	}
	if len(extra) > 0 {
		t.Fatalf("Expected %s to not contain elements %s", slice, extra)
	}
}

func AssertMatch(t *testing.T, actual string, expected string) {
	t.Helper()
	if !regexp.MustCompile(expected).MatchString(actual) {
		t.Fatalf("Expected '%s' to match regex '%s'", actual, expected)
	}
}

// AssertNilE checks for nil value, if not nil it sets test as failed without stopping execution.
func AssertNilE(t *testing.T, actual interface{}) {
	t.Helper()
	if !isNil(actual) {
		t.Errorf("Expected nil: %s", actual)
	}
}

// AssertNil checks for nil value, if not nil it fails the test and stops execution immediately.
func AssertNil(t *testing.T, actual interface{}) {
	t.Helper()
	if !isNil(actual) {
		t.Fatalf("Expected nil: %s", actual)
	}
}

func AssertNotNil(t *testing.T, actual interface{}) {
	t.Helper()
	if isNil(actual) {
		t.Fatal("Expected not nil")
	}
}

func AssertTarball(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(filepath.Clean(path))
	AssertNil(t, err)
	defer f.Close()

	reader := tar.NewReader(f)
	_, err = reader.Next()
	AssertNil(t, err)
}

func isNil(value interface{}) bool {
	return value == nil || (reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil())
}

func hasMatches(actual, exp string) bool {
	regex := regexp.MustCompile(exp)
	matches := regex.FindAll([]byte(actual), -1)
	return len(matches) > 0
}

var dockerCliVal client.CommonAPIClient
var dockerCliOnce sync.Once
var dockerCliErr error

func dockerCli(t *testing.T) client.CommonAPIClient {
	dockerCliOnce.Do(func() {
		dockerCliVal, dockerCliErr = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	})
	AssertNil(t, dockerCliErr)
	return dockerCliVal
}

func Eventually(t *testing.T, test func() bool, every time.Duration, timeout time.Duration) {
	t.Helper()

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if test() {
				return
			}
		case <-timer.C:
			t.Fatalf("timeout on eventually: %v", timeout)
		}
	}
}

func CreateImage(t *testing.T, dockerCli client.CommonAPIClient, repoName, dockerFile string) {
	t.Helper()

	buildContext := archive.CreateSingleFileTarReader("Dockerfile", dockerFile)
	defer buildContext.Close()

	resp, err := dockerCli.ImageBuild(context.Background(), buildContext, dockertypes.ImageBuildOptions{
		Tags:           []string{repoName},
		SuppressOutput: true,
		Remove:         true,
		ForceRemove:    true,
	})
	AssertNil(t, err)

	defer resp.Body.Close()
	err = checkResponse(resp.Body)
	AssertNil(t, errors.Wrapf(err, "building image %s", style.Symbol(repoName)))
}

func CreateImageFromDir(t *testing.T, dockerCli client.CommonAPIClient, repoName string, dir string) {
	t.Helper()

	buildContext := archive.ReadDirAsTar(dir, "/", 0, 0, -1, true, false, nil)
	resp, err := dockerCli.ImageBuild(context.Background(), buildContext, dockertypes.ImageBuildOptions{
		Tags:           []string{repoName},
		Remove:         true,
		ForceRemove:    true,
		SuppressOutput: false,
	})
	AssertNil(t, err)

	defer resp.Body.Close()
	err = checkResponse(resp.Body)
	AssertNil(t, errors.Wrapf(err, "building image %s", style.Symbol(repoName)))
}

func CheckImageBuildResult(response dockertypes.ImageBuildResponse, err error) error {
	if err != nil {
		return err
	}

	defer response.Body.Close()
	return checkResponse(response.Body)
}

func checkResponse(responseBody io.Reader) error {
	body, err := io.ReadAll(responseBody)
	if err != nil {
		return errors.Wrap(err, "reading body")
	}

	messages := strings.Builder{}
	for _, line := range bytes.Split(body, []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		var msg jsonmessage.JSONMessage
		err := json.Unmarshal(line, &msg)
		if err != nil {
			return errors.Wrapf(err, "expected JSON: %s", string(line))
		}

		if msg.Stream != "" {
			messages.WriteString(msg.Stream)
		}

		if msg.Error != nil {
			return errors.WithMessage(msg.Error, messages.String())
		}
	}

	return nil
}

func CreateImageOnRemote(t *testing.T, dockerCli client.CommonAPIClient, registryConfig *TestRegistryConfig, repoName, dockerFile string) string {
	t.Helper()
	imageName := registryConfig.RepoName(repoName)
	CreateImage(t, dockerCli, imageName, dockerFile)
	AssertNil(t, PushImage(dockerCli, imageName, registryConfig))
	return imageName
}

func DockerRmi(dockerCli client.CommonAPIClient, repoNames ...string) error {
	var err error
	ctx := context.Background()
	for _, name := range repoNames {
		_, e := dockerCli.ImageRemove(
			ctx,
			name,
			image.RemoveOptions{Force: true, PruneChildren: true},
		)
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func PushImage(dockerCli client.CommonAPIClient, ref string, registryConfig *TestRegistryConfig) error {
	rc, err := dockerCli.ImagePush(context.Background(), ref, image.PushOptions{RegistryAuth: registryConfig.RegistryAuth()})
	if err != nil {
		return errors.Wrap(err, "pushing image")
	}

	defer rc.Close()
	err = checkResponse(rc)
	if err != nil {
		return errors.Wrap(err, "push response")
	}

	return nil
}

func HTTPGetE(url string, headers map[string]string) (string, error) {
	client := http.DefaultClient

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "making new request")
	}

	for key, val := range headers {
		request.Header.Set(key, val)
	}

	resp, err := client.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "doing request")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP Status was bad: %s => %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading body")
	}
	return string(b), nil
}

func ImageID(t *testing.T, repoName string) string {
	t.Helper()
	inspect, _, err := dockerCli(t).ImageInspectWithRaw(context.Background(), repoName)
	AssertNil(t, err)
	return strings.TrimPrefix(inspect.ID, "sha256:")
}

func Digest(t *testing.T, repoName string) string {
	t.Helper()
	inspect, _, err := dockerCli(t).ImageInspectWithRaw(context.Background(), repoName)
	AssertNil(t, err)
	if len(inspect.RepoDigests) < 1 {
		t.Fatalf("image '%s' has no repo digests", repoName)
	}
	parts := strings.Split(inspect.RepoDigests[0], "@")
	if len(parts) < 2 {
		t.Fatalf("repo digest '%s' malformed", inspect.RepoDigests[0])
	}
	return parts[1]
}

func TopLayerDiffID(t *testing.T, repoName string) string {
	t.Helper()
	inspect, _, err := dockerCli(t).ImageInspectWithRaw(context.Background(), repoName)
	AssertNil(t, err)
	if len(inspect.RootFS.Layers) < 1 {
		t.Fatalf("image '%s' has no layers", repoName)
	}
	return inspect.RootFS.Layers[len(inspect.RootFS.Layers)-1]
}

func Run(t *testing.T, cmd *exec.Cmd) string {
	t.Helper()
	txt, err := RunE(cmd)
	AssertNil(t, err)
	return txt
}

func RunE(cmd *exec.Cmd) (string, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to execute command: %v, %s, %s", cmd.Args, err, output)
	}

	return string(output), nil
}

func PullImageWithAuth(dockerCli client.CommonAPIClient, ref, registryAuth string) error {
	rc, err := dockerCli.ImagePull(context.Background(), ref, image.PullOptions{RegistryAuth: registryAuth})
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, rc); err != nil {
		return err
	}
	return rc.Close()
}

func CopyFile(t *testing.T, src, dst string) {
	t.Helper()

	err := CopyFileE(src, dst)
	AssertNil(t, err)
}

func CopyFileE(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(filepath.Clean(dst), os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	modifiedtime := time.Time{}
	return os.Chtimes(dst, modifiedtime, modifiedtime)
}

func RecursiveCopy(t *testing.T, src, dst string) {
	t.Helper()

	err := RecursiveCopyE(src, dst)
	AssertNil(t, err)
}

func RecursiveCopyE(src, dst string) error {
	fis, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range fis {
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			err = CopyFileE(filepath.Join(src, fi.Name()), filepath.Join(dst, fi.Name()))
			if err != nil {
				return err
			}
		}
		if fi.IsDir() {
			err = os.Mkdir(filepath.Join(dst, fi.Name()), fi.Mode())
			if err != nil {
				return err
			}
			err = RecursiveCopyE(filepath.Join(src, fi.Name()), filepath.Join(dst, fi.Name()))
			if err != nil {
				return err
			}
		}
	}

	modifiedtime := time.Time{}
	err = os.Chtimes(dst, modifiedtime, modifiedtime)
	if err != nil {
		return err
	}

	return os.Chmod(dst, 0775)
}

func RequireDocker(t *testing.T) {
	noDocker := os.Getenv("NO_DOCKER")
	SkipIf(t, strings.ToLower(noDocker) == "true" || noDocker == "1", "Skipping because docker daemon unavailable")
}

func SkipIf(t *testing.T, expression bool, reason string) {
	t.Helper()
	if expression {
		t.Skip(reason)
	}
}

func SkipUnless(t *testing.T, expression bool, reason string) {
	t.Helper()
	if !expression {
		t.Skip(reason)
	}
}

func RunContainer(ctx context.Context, dockerCli client.CommonAPIClient, id string, stdout io.Writer, stderr io.Writer) error {
	bodyChan, errChan := container.ContainerWaitWrapper(ctx, dockerCli, id, dcontainer.WaitConditionNextExit)

	logs, err := dockerCli.ContainerAttach(ctx, id, dcontainer.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return err
	}

	if err := dockerCli.ContainerStart(ctx, id, dcontainer.StartOptions{}); err != nil {
		return errors.Wrap(err, "container start")
	}

	copyErr := make(chan error)
	go func() {
		_, err := stdcopy.StdCopy(stdout, stderr, logs.Reader)
		copyErr <- err
	}()

	select {
	case body := <-bodyChan:
		if body.StatusCode != 0 {
			return fmt.Errorf("failed with status code: %d", body.StatusCode)
		}
	case err := <-errChan:
		return err
	}
	return <-copyErr
}

func CreateTGZ(t *testing.T, srcDir, tarDir string, mode int64) string {
	t.Helper()

	fh, err := os.CreateTemp("", "*.tgz")
	AssertNil(t, err)
	defer fh.Close()

	gw := gzip.NewWriter(fh)
	defer gw.Close()

	writeTAR(t, srcDir, tarDir, mode, gw)

	return fh.Name()
}

func CreateTAR(t *testing.T, srcDir, tarDir string, mode int64) string {
	t.Helper()

	fh, err := os.CreateTemp("", "*.tgz")
	AssertNil(t, err)
	defer fh.Close()

	writeTAR(t, srcDir, tarDir, mode, fh)

	return fh.Name()
}

func writeTAR(t *testing.T, srcDir, tarDir string, mode int64, w io.Writer) {
	t.Helper()
	tw := tar.NewWriter(w)
	defer tw.Close()

	err := archive.WriteDirToTar(tw, srcDir, tarDir, 0, 0, mode, true, false, nil)
	AssertNil(t, err)
}

func RecursiveCopyNow(t *testing.T, src, dst string) {
	t.Helper()
	err := os.MkdirAll(dst, 0750)
	AssertNil(t, err)

	fis, err := os.ReadDir(src)
	AssertNil(t, err)
	for _, entry := range fis {
		fi, err := entry.Info()
		AssertNil(t, err)
		if fi.Mode().IsRegular() {
			srcFile, err := os.Open(filepath.Join(filepath.Clean(src), fi.Name()))
			AssertNil(t, err)
			dstFile, err := os.Create(filepath.Join(dst, fi.Name()))
			AssertNil(t, err)
			_, err = io.Copy(dstFile, srcFile)
			AssertNil(t, err)
			modifiedTime := time.Now().Local()
			err = os.Chtimes(filepath.Join(dst, fi.Name()), modifiedTime, modifiedTime)
			AssertNil(t, err)
			err = os.Chmod(filepath.Join(dst, fi.Name()), 0664)
			AssertNil(t, err)
		}
		if fi.IsDir() {
			err = os.Mkdir(filepath.Join(dst, fi.Name()), fi.Mode())
			AssertNil(t, err)
			RecursiveCopyNow(t, filepath.Join(src, fi.Name()), filepath.Join(dst, fi.Name()))
		}
	}
	modifiedTime := time.Now().Local()
	err = os.Chtimes(dst, modifiedTime, modifiedTime)
	AssertNil(t, err)
	err = os.Chmod(dst, 0775)
	AssertNil(t, err)
}

func AssertTarFileContents(t *testing.T, tarfile, path, expected string) {
	t.Helper()
	exist, contents := tarFileContents(t, tarfile, path)
	if !exist {
		t.Fatalf("%s does not exist in %s", path, tarfile)
	}
	AssertEq(t, contents, expected)
}

func tarFileContents(t *testing.T, tarfile, path string) (exist bool, contents string) {
	t.Helper()
	r, err := os.Open(filepath.Clean(tarfile))
	AssertNil(t, err)
	defer r.Close()

	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		AssertNil(t, err)

		if header.Name == path {
			buf, err := io.ReadAll(tr)
			AssertNil(t, err)
			return true, string(buf)
		}
	}
	return false, ""
}

func AssertTarHasFile(t *testing.T, tarFile, path string) {
	t.Helper()

	exist := tarHasFile(t, tarFile, path)
	if !exist {
		t.Fatalf("%s does not exist in %s", path, tarFile)
	}
}

func tarHasFile(t *testing.T, tarFile, path string) (exist bool) {
	t.Helper()

	r, err := os.Open(filepath.Clean(tarFile))
	AssertNil(t, err)
	defer r.Close()

	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		AssertNil(t, err)

		if header.Name == path {
			return true
		}
	}

	return false
}

func AssertBuildpacksHaveDescriptors(t *testing.T, modules []buildpack.BuildModule, descriptors []dist.BuildpackDescriptor) {
	AssertEq(t, len(modules), len(descriptors))
	for _, mod := range modules {
		found := false
		modDesc, ok := mod.Descriptor().(*dist.BuildpackDescriptor)
		AssertEq(t, ok, true)
		for _, descriptor := range descriptors {
			if diff := cmp.Diff(*modDesc, descriptor); diff == "" {
				found = true
				break
			}
		}
		AssertTrue(t, found)
	}
}

func AssertGitHeadEq(t *testing.T, path1, path2 string) {
	r1, err := git.PlainOpen(path1)
	AssertNil(t, err)

	r2, err := git.PlainOpen(path2)
	AssertNil(t, err)

	h1, err := r1.Head()
	AssertNil(t, err)

	h2, err := r2.Head()
	AssertNil(t, err)

	AssertEq(t, h1.Hash().String(), h2.Hash().String())
}

func AssertBlobsLen(t *testing.T, path string, expected int) {
	t.Helper()
	fis, err := os.ReadDir(filepath.Join(path, "blobs", "sha256"))
	AssertNil(t, err)
	AssertEq(t, len(fis), expected)
}

func MockWriterAndOutput() (*color.Console, func() string) {
	r, w, _ := os.Pipe()
	console := color.NewConsole(w)
	return console, func() string {
		_ = w.Close()
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		_ = r.Close()
		return b.String()
	}
}

func LayerFileName(bp buildpack.BuildModule) string {
	return fmt.Sprintf("%s.%s.tar", bp.Descriptor().Info().ID, bp.Descriptor().Info().Version)
}

type FakeAddedLayerImage struct {
	*fakes.Image
	addedLayersOrder []string
}

func (f *FakeAddedLayerImage) AddedLayersOrder() []string {
	return f.addedLayersOrder
}

func (f *FakeAddedLayerImage) AddLayerWithDiffID(path, diffID string) error {
	f.addedLayersOrder = append(f.addedLayersOrder, path)
	return f.Image.AddLayerWithDiffID(path, diffID)
}
