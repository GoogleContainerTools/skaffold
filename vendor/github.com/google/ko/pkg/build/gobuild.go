// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	gb "go/build"
	"io"
	"log"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/ko/internal/sbom"
	"github.com/google/ko/pkg/caps"
	"github.com/google/ko/pkg/internal/git"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ocimutate "github.com/sigstore/cosign/v2/pkg/oci/mutate"
	"github.com/sigstore/cosign/v2/pkg/oci/signed"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	ctypes "github.com/sigstore/cosign/v2/pkg/types"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/tools/go/packages"
)

const (
	defaultAppFilename = "ko-app"

	defaultGoBin = "go"         // defaults to first go binary found in PATH
	goBinPathEnv = "KO_GO_PATH" // env lookup for optional relative or full go binary path
)

// GetBase takes an importpath and returns a base image reference and base image (or index).
type GetBase func(context.Context, string) (name.Reference, Result, error)

// buildContext provides parameters for a builder function.
type buildContext struct {
	creationTime v1.Time
	ip           string
	dir          string
	env          []string
	flags        []string
	ldflags      []string
	platform     v1.Platform
}

type builder func(context.Context, buildContext) (string, error)

type sbomber func(context.Context, string, string, string, oci.SignedEntity, string) ([]byte, types.MediaType, error)

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
	sbomDir              string
	disableOptimizations bool
	trimpath             bool
	buildConfigs         map[string]Config
	defaultEnv           []string
	defaultFlags         []string
	defaultLdflags       []string
	platformMatcher      *platformMatcher
	dir                  string
	labels               map[string]string
	annotations          map[string]string
	user                 string
	debug                bool
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
	sbomDir              string
	disableOptimizations bool
	trimpath             bool
	buildConfigs         map[string]Config
	defaultEnv           []string
	defaultFlags         []string
	defaultLdflags       []string
	platforms            []string
	labels               map[string]string
	annotations          map[string]string
	user                 string
	dir                  string
	jobs                 int
	debug                bool
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
	if gbo.annotations == nil {
		gbo.annotations = map[string]string{}
	}
	return &gobuild{
		ctx:                  gbo.ctx,
		getBase:              gbo.getBase,
		user:                 gbo.user,
		creationTime:         gbo.creationTime,
		kodataCreationTime:   gbo.kodataCreationTime,
		build:                gbo.build,
		sbom:                 gbo.sbom,
		sbomDir:              gbo.sbomDir,
		disableOptimizations: gbo.disableOptimizations,
		trimpath:             gbo.trimpath,
		buildConfigs:         gbo.buildConfigs,
		defaultEnv:           gbo.defaultEnv,
		defaultFlags:         gbo.defaultFlags,
		defaultLdflags:       gbo.defaultLdflags,
		labels:               gbo.labels,
		annotations:          gbo.annotations,
		dir:                  gbo.dir,
		debug:                gbo.debug,
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

func getGoBinary() string {
	if env := os.Getenv(goBinPathEnv); env != "" {
		return env
	}
	return defaultGoBin
}

func doesPlatformSupportDebugging(platform v1.Platform) bool {
	// Here's the list of supported platforms by Delve:
	//
	// https://github.com/go-delve/delve/blob/master/Documentation/faq.md#unsupportedplatforms
	//
	// For the time being, we'll support only linux/amd64 and linux/arm64.

	return platform.OS == "linux" && (platform.Architecture == "amd64" || platform.Architecture == "arm64")
}

func getDelve(ctx context.Context, platform v1.Platform) (string, error) {
	const delveCloneURL = "https://github.com/go-delve/delve.git"

	if platform.OS == "" || platform.Architecture == "" {
		return "", fmt.Errorf("platform os (%q) or arch (%q) is empty",
			platform.OS,
			platform.Architecture,
		)
	}

	env, err := buildEnv(platform, os.Environ(), nil)
	if err != nil {
		return "", fmt.Errorf("could not create env for Delve build: %w", err)
	}

	tmpInstallDir, err := os.MkdirTemp("", "delve")
	if err != nil {
		return "", fmt.Errorf("could not create tmp dir for Delve installation: %w", err)
	}
	cloneDir := filepath.Join(tmpInstallDir, "delve")
	err = os.MkdirAll(cloneDir, 0755)
	if err != nil {
		return "", fmt.Errorf("making dir for delve clone: %w", err)
	}
	err = git.Clone(ctx, cloneDir, delveCloneURL)
	if err != nil {
		return "", fmt.Errorf("cloning delve repo: %w", err)
	}
	osArchDir := fmt.Sprintf("%s_%s", platform.OS, platform.Architecture)
	delveBinaryPath := filepath.Join(tmpInstallDir, "bin", osArchDir, "dlv")

	// install delve to tmp directory
	args := []string{
		"build",
		"-trimpath",
		"-ldflags=-s -w",
		"-o",
		delveBinaryPath,
		"./cmd/dlv",
	}

	gobin := getGoBinary()
	cmd := exec.CommandContext(ctx, gobin, args...)
	cmd.Env = env
	cmd.Dir = cloneDir

	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	log.Printf("Building Delve for %s", platform)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpInstallDir)
		return "", fmt.Errorf("go build Delve: %w: %s", err, output.String())
	}

	if _, err := os.Stat(delveBinaryPath); err != nil {
		return "", fmt.Errorf("could not find Delve binary at %q: %w", delveBinaryPath, err)
	}

	return delveBinaryPath, nil
}

func build(ctx context.Context, buildCtx buildContext) (string, error) {
	// Create the set of build arguments from the config flags/ldflags with any
	// template parameters applied.
	buildArgs, err := createBuildArgs(ctx, buildCtx)
	if err != nil {
		return "", err
	}

	args := make([]string, 0, 4+len(buildArgs))
	args = append(args, "build")
	args = append(args, buildArgs...)
	tmpDir := ""

	if dir := os.Getenv("KOCACHE"); dir != "" {
		dirInfo, err := os.Stat(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("could not stat KOCACHE: %w", err)
			}
			if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
				return "", fmt.Errorf("could not create KOCACHE dir %s: %w", dir, err)
			}
		} else if !dirInfo.IsDir() {
			return "", fmt.Errorf("KOCACHE should be a directory, %s is not a directory", dir)
		}

		// TODO(#264): if KOCACHE is unset, default to filepath.Join(os.TempDir(), "ko").
		tmpDir = filepath.Join(dir, "bin", buildCtx.ip, buildCtx.platform.String())
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("creating KOCACHE bin dir: %w", err)
		}
	} else {
		tmpDir, err = os.MkdirTemp("", "ko")
		if err != nil {
			return "", err
		}
	}

	file := filepath.Join(tmpDir, "out")

	args = append(args, "-o", file)
	args = append(args, buildCtx.ip)

	gobin := getGoBinary()
	cmd := exec.CommandContext(ctx, gobin, args...)
	cmd.Dir = buildCtx.dir
	cmd.Env = buildCtx.env

	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	log.Printf("Building %s for %s", buildCtx.ip, buildCtx.platform)
	if err := cmd.Run(); err != nil {
		if os.Getenv("KOCACHE") == "" {
			_ = os.RemoveAll(tmpDir)
		}
		return "", fmt.Errorf("go build: %w: %s", err, output.String())
	}
	return file, nil
}

func goenv(ctx context.Context) (map[string]string, error) {
	gobin := getGoBinary()
	cmd := exec.CommandContext(ctx, gobin, "env")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go env: %w: %s", err, stderr.String())
	}

	env := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))

	line := 0
	for scanner.Scan() {
		line++
		kv := strings.SplitN(scanner.Text(), "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("go env: failed parsing line: %d", line)
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		// Unquote the value. Handle single or double quoted strings.
		if len(value) > 1 && ((value[0] == '\'' && value[len(value)-1] == '\'') ||
			(value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		env[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("go env: failed parsing: %w", err)
	}
	return env, nil
}

func goversionm(ctx context.Context, file string, appPath string, appFileName string, se oci.SignedEntity, dir string) ([]byte, types.MediaType, error) {
	gobin := getGoBinary()

	switch se.(type) {
	case oci.SignedImage:
		sbom := bytes.NewBuffer(nil)
		cmd := exec.CommandContext(ctx, gobin, "version", "-m", file)
		cmd.Stdout = sbom
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, "", fmt.Errorf("go version -m %s: %w", file, err)
		}

		// In order to get deterministics SBOMs replace our randomized
		// file name with the path the app will get inside of the container.
		s := []byte(strings.Replace(sbom.String(), file, appPath, 1))

		if err := writeSBOM(s, appFileName, dir, "go.version-m"); err != nil {
			return nil, "", fmt.Errorf("writing sbom: %w", err)
		}

		return s, "application/vnd.go.version-m", nil

	case oci.SignedImageIndex:
		return nil, "", nil

	default:
		return nil, "", fmt.Errorf("unrecognized type: %T", se)
	}
}

func spdx(version string) sbomber {
	return func(ctx context.Context, file string, appPath string, appFileName string, se oci.SignedEntity, dir string) ([]byte, types.MediaType, error) {
		switch obj := se.(type) {
		case oci.SignedImage:
			b, _, err := goversionm(ctx, file, appPath, "", obj, "")
			if err != nil {
				return nil, "", err
			}

			b, err = sbom.GenerateImageSPDX(version, b, obj)
			if err != nil {
				return nil, "", err
			}

			if err := writeSBOM(b, appFileName, dir, "spdx.json"); err != nil {
				return nil, "", err
			}

			return b, ctypes.SPDXJSONMediaType, nil

		case oci.SignedImageIndex:
			b, err := sbom.GenerateIndexSPDX(version, obj)
			if err != nil {
				return nil, "", err
			}

			if err := writeSBOM(b, appFileName, dir, "spdx.json"); err != nil {
				return nil, "", err
			}

			return b, ctypes.SPDXJSONMediaType, err

		default:
			return nil, "", fmt.Errorf("unrecognized type: %T", se)
		}
	}
}

func writeSBOM(sbom []byte, appFileName, dir, ext string) error {
	if dir != "" {
		sbomDir := filepath.Clean(dir)
		if err := os.MkdirAll(sbomDir, os.ModePerm); err != nil {
			return err
		}
		sbomPath := filepath.Join(sbomDir, appFileName+"."+ext)
		log.Printf("Writing SBOM to %s", sbomPath)
		return os.WriteFile(sbomPath, sbom, 0644) //nolint:gosec
	}
	return nil
}

// buildEnv creates the environment variables used by the `go build` command.
// From `os/exec.Cmd`: If there are duplicate environment keys, only the last
// value in the slice for each duplicate key is used.
func buildEnv(platform v1.Platform, osEnv, buildEnv []string) ([]string, error) {
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

	env = append(env, osEnv...)
	env = append(env, buildEnv...)
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

func tarBinary(name, binary string, platform *v1.Platform, opts *layerOptions) (*bytes.Buffer, error) {
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
			return nil, fmt.Errorf("writing dir %q to tar: %w", dir, err)
		}
	}

	file, err := os.Open(binary)
	if err != nil {
		return nil, fmt.Errorf("opening binary: %w", err)
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
		Mode:       0555,
		PAXRecords: map[string]string{},
	}
	switch platform.OS {
	case "windows":
		// This magic value is for some reason needed for Windows to be
		// able to execute the binary.
		header.PAXRecords["MSWINDOWS.rawsd"] = userOwnerAndGroupSID
	case "linux":
		if opts.linuxCapabilities != nil {
			xattr, err := opts.linuxCapabilities.ToXattrBytes()
			if err != nil {
				return nil, fmt.Errorf("caps.FileCaps.ToXattrBytes: %w", err)
			}
			header.PAXRecords["SCHILY.xattr.security.capability"] = string(xattr)
		}
	}
	// write the header to the tarball archive
	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("writing tar header: %w", err)
	}
	// copy the file data to the tarball
	if _, err := io.Copy(tw, file); err != nil {
		return nil, fmt.Errorf("copying file to tar: %w", err)
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

func createTemplateData(ctx context.Context, buildCtx buildContext) (map[string]interface{}, error) {
	envVars := map[string]string{
		"LDFLAGS": "",
	}
	for _, entry := range buildCtx.env {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid environment variable entry: %q", entry)
		}
		envVars[kv[0]] = kv[1]
	}

	// Get the go environment.
	goEnv, err := goenv(ctx)
	if err != nil {
		return nil, err
	}

	// Override go env with any matching values from the environment variables.
	for k, v := range envVars {
		if _, ok := goEnv[k]; ok {
			goEnv[k] = v
		}
	}

	// Get the git information, if available.
	info, err := git.GetInfo(ctx, buildCtx.dir)
	if err != nil {
		log.Printf("%v", err)
	}

	// Use the creation time as the build date, if provided.
	date := buildCtx.creationTime.Time
	if date.IsZero() {
		date = time.Now()
	}

	return map[string]interface{}{
		"Env":       envVars,
		"GoEnv":     goEnv,
		"Git":       info.TemplateValue(),
		"Date":      date.Format(time.RFC3339),
		"Timestamp": date.UTC().Unix(),
	}, nil
}

func applyTemplating(list []string, data map[string]interface{}) ([]string, error) {
	result := make([]string, 0, len(list))
	for _, entry := range list {
		tmpl, err := template.New("argsTmpl").Option("missingkey=error").Parse(entry)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}

		result = append(result, buf.String())
	}

	return result, nil
}

func createBuildArgs(ctx context.Context, buildCtx buildContext) ([]string, error) {
	var args []string

	data, err := createTemplateData(ctx, buildCtx)
	if err != nil {
		return nil, err
	}

	if len(buildCtx.flags) > 0 {
		flags, err := applyTemplating(buildCtx.flags, data)
		if err != nil {
			return nil, err
		}

		args = append(args, flags...)
	}

	if len(buildCtx.ldflags) > 0 {
		ldflags, err := applyTemplating(buildCtx.ldflags, data)
		if err != nil {
			return nil, err
		}

		args = append(args, fmt.Sprintf("-ldflags=%s", strings.Join(ldflags, " ")))
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

func (g gobuild) useDebugging(platform v1.Platform) bool {
	return g.debug && doesPlatformSupportDebugging(platform)
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
		if cf.OS == "" {
			cf.OS = "linux"
		}
		if cf.Architecture == "" {
			cf.Architecture = "amd64"
		}

		platform = &v1.Platform{
			OS:           cf.OS,
			Architecture: cf.Architecture,
			OSVersion:    cf.OSVersion,
		}
	}
	if g.debug && !doesPlatformSupportDebugging(*platform) {
		log.Printf("image for platform %q will be built without debugging enabled because debugging is not supported for that platform", *platform)
	}

	if !g.platformMatcher.matches(platform) {
		return nil, fmt.Errorf("base image platform %q does not match desired platforms %v", platform, g.platformMatcher.platforms)
	}

	config := g.configForImportPath(ref.Path())

	// Merge the system and build environment variables.
	env := config.Env
	if len(env) == 0 {
		// Use the default, if any.
		env = g.defaultEnv
	}
	env, err = buildEnv(*platform, os.Environ(), env)
	if err != nil {
		return nil, fmt.Errorf("could not create env for %s: %w", ref.Path(), err)
	}

	// Get the build flags.
	flags := config.Flags
	if len(flags) == 0 {
		// Use the default, if any.
		flags = g.defaultFlags
	}

	// Get the build ldflags.
	ldflags := config.Ldflags
	if len(ldflags) == 0 {
		// Use the default, if any
		ldflags = g.defaultLdflags
	}

	// Do the build into a temporary file.
	file, err := g.build(ctx, buildContext{
		creationTime: g.creationTime,
		ip:           ref.Path(),
		dir:          g.dir,
		env:          env,
		flags:        flags,
		ldflags:      ldflags,
		platform:     *platform,
	})
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	if os.Getenv("KOCACHE") == "" {
		defer os.RemoveAll(filepath.Dir(file))
	}

	var layers []mutate.Addendum

	// Create a layer from the kodata directory under this import path.
	dataLayerBuf, err := g.tarKoData(ref, platform)
	if err != nil {
		return nil, fmt.Errorf("tarring kodata: %w", err)
	}
	dataLayerBytes := dataLayerBuf.Bytes()
	dataLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(dataLayerBytes)), nil
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
	appFileName := appFilename(ref.Path())
	appPath := path.Join(appDir, appFileName)

	var lo layerOptions
	lo.linuxCapabilities, err = caps.NewFileCaps(config.LinuxCapabilities...)
	if err != nil {
		return nil, fmt.Errorf("linux_capabilities: %w", err)
	}

	miss := func() (v1.Layer, error) {
		return buildLayer(appPath, file, platform, layerMediaType, &lo)
	}

	var binaryLayer v1.Layer
	switch {
	case lo.linuxCapabilities != nil:
		log.Printf("Some options prevent us from using layer cache")
		binaryLayer, err = miss()
	default:
		binaryLayer, err = g.cache.get(ctx, file, miss)
	}
	if err != nil {
		return nil, fmt.Errorf("cache.get(%q): %w", file, err)
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

	delvePath := "" // path for delve in image
	if g.useDebugging(*platform) {
		// get delve locally
		delveBinary, err := getDelve(ctx, *platform)
		if err != nil {
			return nil, fmt.Errorf("building Delve: %w", err)
		}
		defer os.RemoveAll(filepath.Dir(delveBinary))

		delvePath = path.Join("/ko-app", filepath.Base(delveBinary))

		// add layer with delve binary
		delveLayer, err := g.cache.get(ctx, delveBinary, func() (v1.Layer, error) {
			return buildLayer(delvePath, delveBinary, platform, layerMediaType, &lo)
		})
		if err != nil {
			return nil, fmt.Errorf("cache.get(%q): %w", delveBinary, err)
		}

		layers = append(layers, mutate.Addendum{
			Layer:     delveLayer,
			MediaType: layerMediaType,
			History: v1.History{
				Author:    "ko",
				Created:   g.creationTime,
				CreatedBy: "ko build " + ref.String(),
				Comment:   "Delve debugger, at " + delvePath,
			},
		})
	}
	delveArgs := []string{
		"exec",
		"--listen=:40000",
		"--headless",
		"--log",
		"--accept-multiclient",
		"--api-version=2",
		"--",
	}

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
		appPath := `C:\ko-app\` + appFileName
		if g.debug {
			cfg.Config.Entrypoint = append([]string{"C:\\" + delvePath}, delveArgs...)
			cfg.Config.Entrypoint = append(cfg.Config.Entrypoint, appPath)
		} else {
			cfg.Config.Entrypoint = []string{appPath}
		}

		updatePath(cfg, `C:\ko-app`)
		cfg.Config.Env = append(cfg.Config.Env, `KO_DATA_PATH=C:\var\run\ko`)
	} else {
		if g.useDebugging(*platform) {
			cfg.Config.Entrypoint = append([]string{delvePath}, delveArgs...)
			cfg.Config.Entrypoint = append(cfg.Config.Entrypoint, appPath)
		}

		updatePath(cfg, appDir)
		cfg.Config.Env = append(cfg.Config.Env, "KO_DATA_PATH="+kodataRoot)
	}
	cfg.Author = "github.com/ko-build/ko"

	if cfg.Config.Labels == nil {
		cfg.Config.Labels = map[string]string{}
	}
	for k, v := range g.labels {
		cfg.Config.Labels[k] = v
	}

	if g.user != "" {
		cfg.Config.User = g.user
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
		// Construct a path-safe encoding of platform.
		pf := strings.ReplaceAll(strings.ReplaceAll(platform.String(), "/", "-"), ":", "-")
		sbom, mt, err := g.sbom(ctx, file, appPath, fmt.Sprintf("%s-%s", appFileName, pf), si, g.sbomDir)
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

// layerOptions captures additional options to apply when authoring layer
type layerOptions struct {
	linuxCapabilities *caps.FileCaps
}

func buildLayer(appPath, file string, platform *v1.Platform, layerMediaType types.MediaType, opts *layerOptions) (v1.Layer, error) {
	// Construct a tarball with the binary and produce a layer.
	binaryLayerBuf, err := tarBinary(appPath, file, platform, opts)
	if err != nil {
		return nil, fmt.Errorf("tarring binary: %w", err)
	}
	binaryLayerBytes := binaryLayerBuf.Bytes()
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(binaryLayerBytes)), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(layerMediaType))
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
		return nil, fmt.Errorf("fetching base image: %w", err)
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

		annotations := maps.Clone(g.annotations)
		annotations[specsv1.AnnotationBaseImageDigest] = baseDigest.String()
		annotations[specsv1.AnnotationBaseImageName] = baseRef.Name()
		base = mutate.Annotations(base, annotations).(Result)
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

	matches := make([]v1.Descriptor, 0)
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

		annotations := maps.Clone(g.annotations)
		// Decorate the image with the ref of the index, and the matching
		// platform's digest.
		annotations[specsv1.AnnotationBaseImageDigest] = matches[0].Digest.String()
		annotations[specsv1.AnnotationBaseImageName] = baseRef.Name()
		img = mutate.Annotations(img, annotations).(v1.Image)
		return g.buildOne(ctx, ref, img, matches[0].Platform)
	}

	annotations := maps.Clone(g.annotations)
	annotations[specsv1.AnnotationBaseImageName] = baseRef.Name()
	baseDigest, _ := baseIndex.Digest()
	annotations[specsv1.AnnotationBaseImageDigest] = baseDigest.String()

	// Build an image for each matching platform from the base and append
	// it to a new index to produce the result. We use the indices to
	// preserve the base image ordering here.
	errg, gctx := errgroup.WithContext(ctx)
	adds := make([]ocimutate.IndexAddendum, len(matches))
	for i, desc := range matches {
		i, desc := i, desc
		errg.Go(func() error {
			baseImage, err := baseIndex.Image(desc.Digest)
			if err != nil {
				return err
			}

			annotations := maps.Clone(g.annotations)
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
			annotations[specsv1.AnnotationBaseImageDigest] = desc.Digest.String()
			annotations[specsv1.AnnotationBaseImageName] = baseRef.Name()

			baseImage = mutate.Annotations(baseImage, annotations).(v1.Image)

			img, err := g.buildOne(gctx, ref, baseImage, desc.Platform)
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
			annotations).(v1.ImageIndex),
		adds...)

	if g.sbom != nil {
		ref := newRef(ref)
		appFileName := appFilename(ref.Path())
		sbom, mt, err := g.sbom(ctx, "", "", fmt.Sprintf("%s-index", appFileName), idx, g.sbomDir)
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

	platforms := make([]v1.Platform, 0)
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
	// Strip out manifests with "unknown/unknown" platform, which Docker uses
	// to store provenance attestations.
	if base != nil &&
		(base.OS == "unknown" || base.Architecture == "unknown") {
		return false
	}

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
			}
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
		return true
	}

	return false
}
