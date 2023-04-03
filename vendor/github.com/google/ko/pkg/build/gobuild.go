/*
Copyright 2018 Google LLC All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package build

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	gb "go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/containerd/stargz-snapshotter/estargz"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/ko/internal/sbom"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/cosign/pkg/oci"
	ocimutate "github.com/sigstore/cosign/pkg/oci/mutate"
	"github.com/sigstore/cosign/pkg/oci/signed"
	"github.com/sigstore/cosign/pkg/oci/static"
	ctypes "github.com/sigstore/cosign/pkg/types"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/tools/go/packages"
)

const (
	defaultAppFilename = "ko-app"
)

// GetBase takes an importpath and returns a base image reference and base image (or index).
type GetBase func(context.Context, string) (name.Reference, Result, error)

type builder func(context.Context, string, string, v1.Platform, Config) (string, error)

type sbomber func(context.Context, string, string, oci.SignedEntity) ([]byte, types.MediaType, error)

type platformMatcher struct {
	spec      []string
	platforms []v1.Platform
}

type gobuild struct {
	ctx                  context.Context
	getBase              GetBase
	creationTime         v1.Time
	kodataCreationTime   v1.Time
	build                builder
	sbom                 sbomber
	disableOptimizations bool
	trimpath             bool
	buildConfigs         map[string]Config
	platformMatcher      *platformMatcher
	dir                  string
	labels               map[string]string
	semaphore            *semaphore.Weighted

	cache *layerCache
}

// Option is a functional option for NewGo.
type Option func(*gobuildOpener) error

type gobuildOpener struct {
	ctx                  context.Context
	getBase              GetBase
	creationTime         v1.Time
	kodataCreationTime   v1.Time
	build                builder
	sbom                 sbomber
	disableOptimizations bool
	trimpath             bool
	buildConfigs         map[string]Config
	platforms            []string
	labels               map[string]string
	dir                  string
	jobs                 int
}

func (gbo *gobuildOpener) Open() (Interface, error) {
	if gbo.getBase == nil {
		return nil, errors.New("a way of providing base images must be specified, see build.WithBaseImages")
	}
	matcher, err := parseSpec(gbo.platforms)
	if err != nil {
		return nil, err
	}
	if gbo.jobs == 0 {
		gbo.jobs = runtime.GOMAXPROCS(0)
	}
	return &gobuild{
		ctx:                  gbo.ctx,
		getBase:              gbo.getBase,
		creationTime:         gbo.creationTime,
		kodataCreationTime:   gbo.kodataCreationTime,
		build:                gbo.build,
		sbom:                 gbo.sbom,
		disableOptimizations: gbo.disableOptimizations,
		trimpath:             gbo.trimpath,
		buildConfigs:         gbo.buildConfigs,
		labels:               gbo.labels,
		dir:                  gbo.dir,
		platformMatcher:      matcher,
		cache: &layerCache{
			buildToDiff: map[string]buildIDToDiffID{},
			diffToDesc:  map[string]diffIDToDescriptor{},
		},
		semaphore: semaphore.NewWeighted(int64(gbo.jobs)),
	}, nil
}

// NewGo returns a build.Interface implementation that:
//  1. builds go binaries named by importpath,
//  2. containerizes the binary on a suitable base.
//
// The `dir` argument is the working directory for executing the `go` tool.
// If `dir` is empty, the function uses the current process working directory.
func NewGo(ctx context.Context, dir string, options ...Option) (Interface, error) {
	gbo := &gobuildOpener{
		ctx:   ctx,
		build: build,
		dir:   dir,
		sbom:  spdx("(none)"),
	}

	for _, option := range options {
		if err := option(gbo); err != nil {
			return nil, err
		}
	}
	return gbo.Open()
}

func (g *gobuild) qualifyLocalImport(importpath string) (string, error) {
	dir := filepath.Clean(g.dir)
	if dir == "." {
		dir = ""
	}
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, importpath)
	if err != nil {
		return "", err
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf("found %d local packages, expected 1", len(pkgs))
	}
	return pkgs[0].PkgPath, nil
}

// QualifyImport implements build.Interface
func (g *gobuild) QualifyImport(importpath string) (string, error) {
	if gb.IsLocalImport(importpath) {
		var err error
		importpath, err = g.qualifyLocalImport(importpath)
		if err != nil {
			return "", fmt.Errorf("qualifying local import %s: %w", importpath, err)
		}
	}
	if !strings.HasPrefix(importpath, StrictScheme) {
		importpath = StrictScheme + importpath
	}
	return importpath, nil
}

// IsSupportedReference implements build.Interface
//
// Only valid importpaths that provide commands (i.e., are "package main") are
// supported.
func (g *gobuild) IsSupportedReference(s string) error {
	ref := newRef(s)
	if !ref.IsStrict() {
		return errors.New("importpath does not start with ko://")
	}
	dir := filepath.Clean(g.dir)
	if dir == "." {
		dir = ""
	}
	pkgs, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName}, ref.Path())
	if err != nil {
		return fmt.Errorf("error loading package from %s: %w", ref.Path(), err)
	}
	if len(pkgs) != 1 {
		return fmt.Errorf("found %d local packages, expected 1", len(pkgs))
	}
	if pkgs[0].Name != "main" {
		return errors.New("importpath is not `package main`")
	}
	return nil
}

func getGoarm(platform v1.Platform) (string, error) {
	if !strings.HasPrefix(platform.Variant, "v") {
		return "", fmt.Errorf("strange arm variant: %v", platform.Variant)
	}

	vs := strings.TrimPrefix(platform.Variant, "v")
	variant, err := strconv.Atoi(vs)
	if err != nil {
		return "", fmt.Errorf("cannot parse arm variant %q: %w", platform.Variant, err)
	}
	if variant >= 5 {
		// TODO(golang/go#29373): Allow for 8 in later go versions if this is fixed.
		if variant > 7 {
			vs = "7"
		}
		return vs, nil
	}
	return "", nil
}

func build(ctx context.Context, ip string, dir string, platform v1.Platform, config Config) (string, error) {
	buildArgs, err := createBuildArgs(config)
	if err != nil {
		return "", err
	}

	args := make([]string, 0, 4+len(buildArgs))
	args = append(args, "build")
	args = append(args, buildArgs...)

	env, err := buildEnv(platform, os.Environ(), config.Env)
	if err != nil {
		return "", fmt.Errorf("could not create env for %s: %w", ip, err)
	}

	tmpDir, err := ioutil.TempDir("", "ko")
	if err != nil {
		return "", err
	}

	if dir := os.Getenv("KOCACHE"); dir != "" {
		dirInfo, err := os.Stat(dir)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
				return "", fmt.Errorf("could not create KOCACHE dir %s: %w", dir, err)
			}
		} else if !dirInfo.IsDir() {
			return "", fmt.Errorf("KOCACHE should be a directory, %s is not a directory", dir)
		}

		// TODO(#264): if KOCACHE is unset, default to filepath.Join(os.TempDir(), "ko").
		tmpDir = filepath.Join(dir, "bin", ip, platform.String())
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			return "", err
		}
	}

	file := filepath.Join(tmpDir, "out")

	args = append(args, "-o", file)
	args = append(args, ip)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = env

	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	log.Printf("Building %s for %s", ip, platform)
	if err := cmd.Run(); err != nil {
		if os.Getenv("KOCACHE") == "" {
			os.RemoveAll(tmpDir)
		}
		log.Printf("Unexpected error running \"go build\": %v\n%v", err, output.String())
		return "", err
	}
	return file, nil
}

func goversionm(ctx context.Context, file string, appPath string, se oci.SignedEntity) ([]byte, types.MediaType, error) {
	switch se.(type) {
	case oci.SignedImage:
		sbom := bytes.NewBuffer(nil)
		cmd := exec.CommandContext(ctx, "go", "version", "-m", file)
		cmd.Stdout = sbom
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, "", err
		}

		// In order to get deterministics SBOMs replace our randomized
		// file name with the path the app will get inside of the container.
		return []byte(strings.Replace(sbom.String(), file, appPath, 1)), "application/vnd.go.version-m", nil

	case oci.SignedImageIndex:
		return nil, "", nil

	default:
		return nil, "", fmt.Errorf("unrecognized type: %T", se)
	}
}

func spdx(version string) sbomber {
	return func(ctx context.Context, file string, appPath string, se oci.SignedEntity) ([]byte, types.MediaType, error) {
		switch obj := se.(type) {
		case oci.SignedImage:
			b, _, err := goversionm(ctx, file, appPath, obj)
			if err != nil {
				return nil, "", err
			}

			b, err = sbom.GenerateImageSPDX(version, b, obj)
			if err != nil {
				return nil, "", err
			}
			return b, ctypes.SPDXJSONMediaType, nil

		case oci.SignedImageIndex:
			b, err := sbom.GenerateIndexSPDX(version, obj)
			return b, ctypes.SPDXJSONMediaType, err

		default:
			return nil, "", fmt.Errorf("unrecognized type: %T", se)
		}
	}
}

func cycloneDX() sbomber {
	return func(ctx context.Context, file string, appPath string, se oci.SignedEntity) ([]byte, types.MediaType, error) {
		switch obj := se.(type) {
		case oci.SignedImage:
			b, _, err := goversionm(ctx, file, appPath, obj)
			if err != nil {
				return nil, "", err
			}

			b, err = sbom.GenerateImageCycloneDX(b)
			if err != nil {
				return nil, "", err
			}
			return b, ctypes.CycloneDXJSONMediaType, nil

		case oci.SignedImageIndex:
			b, err := sbom.GenerateIndexCycloneDX(obj)
			return b, ctypes.SPDXJSONMediaType, err

		default:
			return nil, "", fmt.Errorf("unrecognized type: %T", se)
		}
	}
}

// buildEnv creates the environment variables used by the `go build` command.
// From `os/exec.Cmd`: If Env contains duplicate environment keys, only the last
// value in the slice for each duplicate key is used.
func buildEnv(platform v1.Platform, userEnv, configEnv []string) ([]string, error) {
	// Default env
	env := []string{
		"CGO_ENABLED=0",
		"GOOS=" + platform.OS,
		"GOARCH=" + platform.Architecture,
	}
	if platform.Variant != "" {
		switch platform.Architecture {
		case "arm":
			// See: https://pkg.go.dev/cmd/go#hdr-Environment_variables
			goarm, err := getGoarm(platform)
			if err != nil {
				return nil, fmt.Errorf("goarm failure: %w", err)
			}
			if goarm != "" {
				env = append(env, "GOARM="+goarm)
			}
		case "amd64":
			// See: https://tip.golang.org/doc/go1.18#amd64
			env = append(env, "GOAMD64="+platform.Variant)
		}
	}

	env = append(env, userEnv...)
	env = append(env, configEnv...)
	return env, nil
}

func appFilename(importpath string) string {
	base := filepath.Base(importpath)

	// If we fail to determine a good name from the importpath then use a
	// safe default.
	if base == "." || base == string(filepath.Separator) {
		return defaultAppFilename
	}

	return base
}

// userOwnerAndGroupSID is a magic value needed to make the binary executable
// in a Windows container.
//
// owner: BUILTIN/Users group: BUILTIN/Users ($sddlValue="O:BUG:BU")
const userOwnerAndGroupSID = "AQAAgBQAAAAkAAAAAAAAAAAAAAABAgAAAAAABSAAAAAhAgAAAQIAAAAAAAUgAAAAIQIAAA=="

func tarBinary(name, binary string, platform *v1.Platform) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Write the parent directories to the tarball archive.
	// For Windows, the layer must contain a Hives/ directory, and the root
	// of the actual filesystem goes in a Files/ directory.
	// For Linux, the binary goes into /ko-app/
	dirs := []string{"ko-app"}
	if platform.OS == "windows" {
		dirs = []string{
			"Hives",
			"Files",
			"Files/ko-app",
		}
		name = "Files" + name
	}
	for _, dir := range dirs {
		if err := tw.WriteHeader(&tar.Header{
			Name:     dir,
			Typeflag: tar.TypeDir,
			// Use a fixed Mode, so that this isn't sensitive to the directory and umask
			// under which it was created. Additionally, windows can only set 0222,
			// 0444, or 0666, none of which are executable.
			Mode: 0555,
		}); err != nil {
			return nil, fmt.Errorf("writing dir %q: %w", dir, err)
		}
	}

	file, err := os.Open(binary)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	header := &tar.Header{
		Name:     name,
		Size:     stat.Size(),
		Typeflag: tar.TypeReg,
		// Use a fixed Mode, so that this isn't sensitive to the directory and umask
		// under which it was created. Additionally, windows can only set 0222,
		// 0444, or 0666, none of which are executable.
		Mode: 0555,
	}
	if platform.OS == "windows" {
		// This magic value is for some reason needed for Windows to be
		// able to execute the binary.
		header.PAXRecords = map[string]string{
			"MSWINDOWS.rawsd": userOwnerAndGroupSID,
		}
	}
	// write the header to the tarball archive
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	// copy the file data to the tarball
	if _, err := io.Copy(tw, file); err != nil {
		return nil, err
	}

	return buf, nil
}

func (g *gobuild) kodataPath(ref reference) (string, error) {
	dir := filepath.Clean(g.dir)
	if dir == "." {
		dir = ""
	}
	pkgs, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedFiles}, ref.Path())
	if err != nil {
		return "", fmt.Errorf("error loading package from %s: %w", ref.Path(), err)
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf("found %d local packages, expected 1", len(pkgs))
	}
	if len(pkgs[0].GoFiles) == 0 {
		return "", fmt.Errorf("package %s contains no Go files", pkgs[0])
	}
	return filepath.Join(filepath.Dir(pkgs[0].GoFiles[0]), "kodata"), nil
}

// Where kodata lives in the image.
const kodataRoot = "/var/run/ko"

// walkRecursive performs a filepath.Walk of the given root directory adding it
// to the provided tar.Writer with root -> chroot.  All symlinks are dereferenced,
// which is what leads to recursion when we encounter a directory symlink.
func walkRecursive(tw *tar.Writer, root, chroot string, creationTime v1.Time, platform *v1.Platform) error {
	return filepath.Walk(root, func(hostPath string, info os.FileInfo, err error) error {
		if hostPath == root {
			return nil
		}
		if err != nil {
			return fmt.Errorf("filepath.Walk(%q): %w", root, err)
		}
		// Skip other directories.
		if info.Mode().IsDir() {
			return nil
		}
		newPath := path.Join(chroot, filepath.ToSlash(hostPath[len(root):]))

		// Don't chase symlinks on Windows, where cross-compiled symlink support is not possible.
		if platform.OS == "windows" {
			if info.Mode()&os.ModeSymlink != 0 {
				log.Println("skipping symlink in kodata for windows:", info.Name())
				return nil
			}
		}

		evalPath, err := filepath.EvalSymlinks(hostPath)
		if err != nil {
			return fmt.Errorf("filepath.EvalSymlinks(%q): %w", hostPath, err)
		}

		// Chase symlinks.
		info, err = os.Stat(evalPath)
		if err != nil {
			return fmt.Errorf("os.Stat(%q): %w", evalPath, err)
		}
		// Skip other directories.
		if info.Mode().IsDir() {
			return walkRecursive(tw, evalPath, newPath, creationTime, platform)
		}

		// Open the file to copy it into the tarball.
		file, err := os.Open(evalPath)
		if err != nil {
			return fmt.Errorf("os.Open(%q): %w", evalPath, err)
		}
		defer file.Close()

		// Copy the file into the image tarball.
		header := &tar.Header{
			Name:     newPath,
			Size:     info.Size(),
			Typeflag: tar.TypeReg,
			// Use a fixed Mode, so that this isn't sensitive to the directory and umask
			// under which it was created. Additionally, windows can only set 0222,
			// 0444, or 0666, none of which are executable.
			Mode:    0555,
			ModTime: creationTime.Time,
		}
		if platform.OS == "windows" {
			// This magic value is for some reason needed for Windows to be
			// able to execute the binary.
			header.PAXRecords = map[string]string{
				"MSWINDOWS.rawsd": userOwnerAndGroupSID,
			}
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar.Writer.WriteHeader(%q): %w", newPath, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("io.Copy(%q, %q): %w", newPath, evalPath, err)
		}
		return nil
	})
}

func (g *gobuild) tarKoData(ref reference, platform *v1.Platform) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	root, err := g.kodataPath(ref)
	if err != nil {
		return nil, err
	}

	creationTime := g.kodataCreationTime

	// Write the parent directories to the tarball archive.
	// For Windows, the layer must contain a Hives/ directory, and the root
	// of the actual filesystem goes in a Files/ directory.
	// For Linux, kodata starts at /var/run/ko.
	chroot := kodataRoot
	dirs := []string{
		"/var",
		"/var/run",
		"/var/run/ko",
	}
	if platform.OS == "windows" {
		chroot = "Files" + kodataRoot
		dirs = []string{
			"Hives",
			"Files",
			"Files/var",
			"Files/var/run",
			"Files/var/run/ko",
		}
	}
	for _, dir := range dirs {
		if err := tw.WriteHeader(&tar.Header{
			Name:     dir,
			Typeflag: tar.TypeDir,
			// Use a fixed Mode, so that this isn't sensitive to the directory and umask
			// under which it was created. Additionally, windows can only set 0222,
			// 0444, or 0666, none of which are executable.
			Mode:    0555,
			ModTime: creationTime.Time,
		}); err != nil {
			return nil, fmt.Errorf("writing dir %q: %w", dir, err)
		}
	}

	return buf, walkRecursive(tw, root, chroot, creationTime, platform)
}

func createTemplateData() map[string]interface{} {
	envVars := map[string]string{
		"LDFLAGS": "",
	}
	for _, entry := range os.Environ() {
		kv := strings.SplitN(entry, "=", 2)
		envVars[kv[0]] = kv[1]
	}

	return map[string]interface{}{
		"Env": envVars,
	}
}

func applyTemplating(list []string, data map[string]interface{}) error {
	for i, entry := range list {
		tmpl, err := template.New("argsTmpl").Option("missingkey=error").Parse(entry)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return err
		}

		list[i] = buf.String()
	}

	return nil
}

func createBuildArgs(buildCfg Config) ([]string, error) {
	var args []string

	data := createTemplateData()

	if len(buildCfg.Flags) > 0 {
		if err := applyTemplating(buildCfg.Flags, data); err != nil {
			return nil, err
		}

		args = append(args, buildCfg.Flags...)
	}

	if len(buildCfg.Ldflags) > 0 {
		if err := applyTemplating(buildCfg.Ldflags, data); err != nil {
			return nil, err
		}

		args = append(args, fmt.Sprintf("-ldflags=%s", strings.Join(buildCfg.Ldflags, " ")))
	}

	// Reject any flags that attempt to set --toolexec (with or
	// without =, with one or two -s)
	for _, a := range args {
		for _, d := range []string{"-", "--"} {
			if a == d+"toolexec" || strings.HasPrefix(a, d+"toolexec=") {
				return nil, fmt.Errorf("cannot set %s", a)
			}
		}
	}

	return args, nil
}

func (g *gobuild) configForImportPath(ip string) Config {
	config := g.buildConfigs[ip]
	if g.trimpath {
		// The `-trimpath` flag removes file system paths from the resulting binary, to aid reproducibility.
		// Ref: https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies
		config.Flags = append(config.Flags, "-trimpath")
	}

	if g.disableOptimizations {
		// Disable optimizations (-N) and inlining (-l).
		config.Flags = append(config.Flags, "-gcflags", "all=-N -l")
	}

	if config.ID != "" {
		log.Printf("Using build config %s for %s", config.ID, ip)
	}

	return config
}

func (g *gobuild) buildOne(ctx context.Context, refStr string, base v1.Image, platform *v1.Platform) (oci.SignedImage, error) {
	if err := g.semaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer g.semaphore.Release(1)

	ref := newRef(refStr)

	// Layers should be typed to match the underlying image, since some
	// registries reject mixed-type layers.
	var layerMediaType types.MediaType
	mt, err := base.MediaType()
	if err != nil {
		return nil, err
	}
	switch mt {
	case types.OCIManifestSchema1:
		layerMediaType = types.OCILayer
	case types.DockerManifestSchema2:
		layerMediaType = types.DockerLayer
	}

	cf, err := base.ConfigFile()
	if err != nil {
		return nil, err
	}
	if platform == nil {
		platform = &v1.Platform{
			OS:           cf.OS,
			Architecture: cf.Architecture,
			OSVersion:    cf.OSVersion,
		}
	}

	if !g.platformMatcher.matches(platform) {
		return nil, fmt.Errorf("base image platform %q does not match desired platforms %v", platform, g.platformMatcher.platforms)
	}
	// Do the build into a temporary file.
	file, err := g.build(ctx, ref.Path(), g.dir, *platform, g.configForImportPath(ref.Path()))
	if err != nil {
		return nil, err
	}
	if os.Getenv("KOCACHE") == "" {
		defer os.RemoveAll(filepath.Dir(file))
	}

	var layers []mutate.Addendum

	// Create a layer from the kodata directory under this import path.
	dataLayerBuf, err := g.tarKoData(ref, platform)
	if err != nil {
		return nil, err
	}
	dataLayerBytes := dataLayerBuf.Bytes()
	dataLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(dataLayerBytes)), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(layerMediaType))
	if err != nil {
		return nil, err
	}
	layers = append(layers, mutate.Addendum{
		Layer: dataLayer,
		History: v1.History{
			Author:    "ko",
			CreatedBy: "ko build " + ref.String(),
			Created:   g.kodataCreationTime,
			Comment:   "kodata contents, at $KO_DATA_PATH",
		},
	})

	appDir := "/ko-app"
	appPath := path.Join(appDir, appFilename(ref.Path()))

	miss := func() (v1.Layer, error) {
		return buildLayer(appPath, file, platform, layerMediaType)
	}

	binaryLayer, err := g.cache.get(ctx, file, miss)
	if err != nil {
		return nil, err
	}

	layers = append(layers, mutate.Addendum{
		Layer:     binaryLayer,
		MediaType: layerMediaType,
		History: v1.History{
			Author:    "ko",
			Created:   g.creationTime,
			CreatedBy: "ko build " + ref.String(),
			Comment:   "go build output, at " + appPath,
		},
	})

	// Augment the base image with our application layer.
	withApp, err := mutate.Append(base, layers...)
	if err != nil {
		return nil, err
	}

	// Start from a copy of the base image's config file, and set
	// the entrypoint to our app.
	cfg, err := withApp.ConfigFile()
	if err != nil {
		return nil, err
	}

	cfg = cfg.DeepCopy()
	cfg.Config.Entrypoint = []string{appPath}
	cfg.Config.Cmd = nil
	if platform.OS == "windows" {
		cfg.Config.Entrypoint = []string{`C:\ko-app\` + appFilename(ref.Path())}
		updatePath(cfg, `C:\ko-app`)
		cfg.Config.Env = append(cfg.Config.Env, `KO_DATA_PATH=C:\var\run\ko`)
	} else {
		updatePath(cfg, appDir)
		cfg.Config.Env = append(cfg.Config.Env, "KO_DATA_PATH="+kodataRoot)
	}
	cfg.Author = "github.com/google/ko"

	if cfg.Config.Labels == nil {
		cfg.Config.Labels = map[string]string{}
	}
	for k, v := range g.labels {
		cfg.Config.Labels[k] = v
	}

	empty := v1.Time{}
	if g.creationTime != empty {
		cfg.Created = g.creationTime
	}

	image, err := mutate.ConfigFile(withApp, cfg)
	if err != nil {
		return nil, err
	}

	si := signed.Image(image)

	if g.sbom != nil {
		sbom, mt, err := g.sbom(ctx, file, appPath, si)
		if err != nil {
			return nil, err
		}
		f, err := static.NewFile(sbom, static.WithLayerMediaType(mt))
		if err != nil {
			return nil, err
		}
		si, err = ocimutate.AttachFileToImage(si, "sbom", f)
		if err != nil {
			return nil, err
		}
	}
	return si, nil
}

func buildLayer(appPath, file string, platform *v1.Platform, layerMediaType types.MediaType) (v1.Layer, error) {
	// Construct a tarball with the binary and produce a layer.
	binaryLayerBuf, err := tarBinary(appPath, file, platform)
	if err != nil {
		return nil, err
	}
	binaryLayerBytes := binaryLayerBuf.Bytes()
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(binaryLayerBytes)), nil
	}, tarball.WithCompressedCaching, tarball.WithEstargzOptions(estargz.WithPrioritizedFiles([]string{
		// When using estargz, prioritize downloading the binary entrypoint.
		appPath,
	})), tarball.WithMediaType(layerMediaType))
}

// Append appPath to the PATH environment variable, if it exists. Otherwise,
// set the PATH environment variable to appPath.
func updatePath(cf *v1.ConfigFile, appPath string) {
	for i, env := range cf.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			// Expect environment variables to be in the form KEY=VALUE, so this is unexpected.
			continue
		}
		key, value := parts[0], parts[1]
		if key == "PATH" {
			value = fmt.Sprintf("%s:%s", value, appPath)
			cf.Config.Env[i] = "PATH=" + value
			return
		}
	}

	// If we get here, we never saw PATH.
	cf.Config.Env = append(cf.Config.Env, "PATH="+appPath)
}

// Build implements build.Interface
func (g *gobuild) Build(ctx context.Context, s string) (Result, error) {
	// Determine the appropriate base image for this import path.
	// We use the overall gobuild.ctx because the Build ctx gets cancelled
	// early, and we lazily use the ctx within ggcr's remote package.
	baseRef, base, err := g.getBase(g.ctx, s)
	if err != nil {
		return nil, err
	}

	// Determine what kind of base we have and if we should publish an image or an index.
	mt, err := base.MediaType()
	if err != nil {
		return nil, err
	}

	// Annotate the base image we pass to the build function with
	// annotations indicating the digest (and possibly tag) of the
	// base image.  This will be inherited by the image produced.
	if mt != types.DockerManifestList {
		baseDigest, err := base.Digest()
		if err != nil {
			return nil, err
		}

		anns := map[string]string{
			specsv1.AnnotationBaseImageDigest: baseDigest.String(),
			specsv1.AnnotationBaseImageName:   baseRef.Name(),
		}
		base = mutate.Annotations(base, anns).(Result)
	}

	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		baseIndex, ok := base.(v1.ImageIndex)
		if !ok {
			return nil, fmt.Errorf("failed to interpret base as index: %v", base)
		}
		return g.buildAll(ctx, s, baseRef, baseIndex)
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		baseImage, ok := base.(v1.Image)
		if !ok {
			return nil, fmt.Errorf("failed to interpret base as image: %v", base)
		}
		return g.buildOne(ctx, s, baseImage, nil)
	default:
		return nil, fmt.Errorf("base image media type: %s", mt)
	}
}

func (g *gobuild) buildAll(ctx context.Context, ref string, baseRef name.Reference, baseIndex v1.ImageIndex) (Result, error) {
	im, err := baseIndex.IndexManifest()
	if err != nil {
		return nil, err
	}

	matches := []v1.Descriptor{}
	for _, desc := range im.Manifests {
		// Nested index is pretty rare. We could support this in theory, but return an error for now.
		if desc.MediaType != types.OCIManifestSchema1 && desc.MediaType != types.DockerManifestSchema2 {
			return nil, fmt.Errorf("%q has unexpected mediaType %q in base for %q", desc.Digest, desc.MediaType, ref)
		}

		if g.platformMatcher.matches(desc.Platform) {
			matches = append(matches, desc)
		}
	}
	if len(matches) == 0 {
		return nil, errors.New("no matching platforms in base image index")
	}
	if len(matches) == 1 {
		// Filters resulted in a single matching platform; just produce
		// a single-platform image.
		img, err := baseIndex.Image(matches[0].Digest)
		if err != nil {
			return nil, fmt.Errorf("error getting matching image from index: %w", err)
		}
		// Decorate the image with the ref of the index, and the matching
		// platform's digest.
		img = mutate.Annotations(img, map[string]string{
			specsv1.AnnotationBaseImageDigest: matches[0].Digest.String(),
			specsv1.AnnotationBaseImageName:   baseRef.Name(),
		}).(v1.Image)
		return g.buildOne(ctx, ref, img, matches[0].Platform)
	}

	// Build an image for each matching platform from the base and append
	// it to a new index to produce the result. We use the indices to
	// preserve the base image ordering here.
	errg, ctx := errgroup.WithContext(ctx)
	adds := make([]ocimutate.IndexAddendum, len(matches))
	for i, desc := range matches {
		i, desc := i, desc
		errg.Go(func() error {
			baseImage, err := baseIndex.Image(desc.Digest)
			if err != nil {
				return err
			}

			// Decorate the image with the ref of the index, and the matching
			// platform's digest.  The ref of the index encodes the critical
			// repository information for fetching the base image's digest, but
			// we leave `name` pointing at the index's full original ref to that
			// folks could conceivably check for updates to the index over time.
			// While the `digest` doesn't give us enough information to check
			// for changes with a simple HEAD (because we need the full index
			// manifest to get the per-architecture image), that optimization
			// mainly matters for DockerHub where HEAD's are exempt from rate
			// limiting.  However, in practice, the way DockerHub updates the
			// indices for official images is to rebuild per-arch images and
			// replace the per-arch images in the existing index, so an index
			// with N manifest receives N updates.  If we only record the digest
			// of the index here, then we cannot tell when the index updates are
			// no-ops for us because we didn't record the digest of the actual
			// image we used, and we would potentially end up doing Nx more work
			// than we really need to do.
			baseImage = mutate.Annotations(baseImage, map[string]string{
				specsv1.AnnotationBaseImageDigest: desc.Digest.String(),
				specsv1.AnnotationBaseImageName:   baseRef.Name(),
			}).(v1.Image)

			img, err := g.buildOne(ctx, ref, baseImage, desc.Platform)
			if err != nil {
				return err
			}
			adds[i] = ocimutate.IndexAddendum{
				Add: img,
				Descriptor: v1.Descriptor{
					URLs:        desc.URLs,
					MediaType:   desc.MediaType,
					Annotations: desc.Annotations,
					Platform:    desc.Platform,
				},
			}
			return nil
		})
	}
	if err := errg.Wait(); err != nil {
		return nil, err
	}

	baseType, err := baseIndex.MediaType()
	if err != nil {
		return nil, err
	}

	idx := ocimutate.AppendManifests(
		mutate.Annotations(
			mutate.IndexMediaType(empty.Index, baseType),
			im.Annotations).(v1.ImageIndex),
		adds...)

	if g.sbom != nil {
		sbom, mt, err := g.sbom(ctx, "", "", idx)
		if err != nil {
			return nil, err
		}
		if sbom != nil {
			f, err := static.NewFile(sbom, static.WithLayerMediaType(mt))
			if err != nil {
				return nil, err
			}
			idx, err = ocimutate.AttachFileToImageIndex(idx, "sbom", f)
			if err != nil {
				return nil, err
			}
		}
	}

	return idx, nil
}

func parseSpec(spec []string) (*platformMatcher, error) {
	// Don't bother parsing "all".
	// Empty slice should never happen because we default to linux/amd64 (or GOOS/GOARCH).
	if len(spec) == 0 || spec[0] == "all" {
		return &platformMatcher{spec: spec}, nil
	}

	platforms := []v1.Platform{}
	for _, s := range spec {
		p, err := v1.ParsePlatform(s)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, *p)
	}
	return &platformMatcher{spec: spec, platforms: platforms}, nil
}

func (pm *platformMatcher) matches(base *v1.Platform) bool {
	if len(pm.spec) > 0 && pm.spec[0] == "all" {
		return true
	}

	// Don't build anything without a platform field unless "all". Unclear what we should do here.
	if base == nil {
		return false
	}

	for _, p := range pm.platforms {
		if p.OS != "" && base.OS != p.OS {
			continue
		}
		if p.Architecture != "" && base.Architecture != p.Architecture {
			continue
		}
		if p.Variant != "" && base.Variant != p.Variant {
			continue
		}

		// Windows is... weird. Windows base images use osversion to
		// communicate what Windows version is used, which matters for image
		// selection at runtime.
		//
		// Windows osversions include the usual major/minor/patch version
		// components, as well as an incrementing "build number" which can
		// change when new Windows base images are released.
		//
		// In order to avoid having to match the entire osversion including the
		// incrementing build number component, we allow matching a platform
		// that only matches the first three osversion components, only for
		// Windows images.
		//
		// If the X.Y.Z components don't match (or aren't formed as we expect),
		// the platform doesn't match. Only if X.Y.Z matches and the extra
		// build number component doesn't, do we consider the platform to
		// match.
		//
		// Ref: https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility?tabs=windows-server-2022%2Cwindows-10-21H1#build-number-new-release-of-windows
		if p.OSVersion != "" && p.OSVersion != base.OSVersion {
			if p.OS != "windows" {
				// osversion mismatch is only possibly allowed when os == windows.
				continue
			} else {
				if pcount, bcount := strings.Count(p.OSVersion, "."), strings.Count(base.OSVersion, "."); pcount == 2 && bcount == 3 {
					if p.OSVersion != base.OSVersion[:strings.LastIndex(base.OSVersion, ".")] {
						// If requested osversion is X.Y.Z and potential match is X.Y.Z.A, all of X.Y.Z must match.
						// Any other form of these osversions are not a match.
						continue
					}
				} else {
					// Partial osversion matching only allows X.Y.Z to match X.Y.Z.A.
					continue
				}
			}
		}
		return true
	}

	return false
}
