package buildpack

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/dist"
)

const (
	KindBuildpack = "buildpack"
	KindExtension = "extension"
)

//go:generate mockgen -package testmocks -destination ../testmocks/mock_build_module.go github.com/buildpacks/pack/pkg/buildpack BuildModule
type BuildModule interface {
	// Open returns a reader to a tar with contents structured as per the distribution spec
	// (currently '/cnb/buildpacks/{ID}/{version}/*', all entries with a zeroed-out
	// timestamp and root UID/GID).
	Open() (io.ReadCloser, error)
	Descriptor() Descriptor
}

type Descriptor interface {
	API() *api.Version
	EnsureStackSupport(stackID string, providedMixins []string, validateRunStageMixins bool) error
	EnsureTargetSupport(os, arch, distroName, distroVersion string) error
	EscapedID() string
	Info() dist.ModuleInfo
	Kind() string
	Order() dist.Order
	Stacks() []dist.Stack
	Targets() []dist.Target
}

type Blob interface {
	// Open returns a io.ReadCloser for the contents of the Blob in tar format.
	Open() (io.ReadCloser, error)
}

type buildModule struct {
	descriptor Descriptor
	Blob       `toml:"-"`
}

func (b *buildModule) Descriptor() Descriptor {
	return b.descriptor
}

// FromBlob constructs a buildpack or extension from a blob. It is assumed that the buildpack
// contents are structured as per the distribution spec (currently '/cnb/buildpacks/{ID}/{version}/*' or
// '/cnb/extensions/{ID}/{version}/*').
func FromBlob(descriptor Descriptor, blob Blob) BuildModule {
	return &buildModule{
		Blob:       blob,
		descriptor: descriptor,
	}
}

// FromBuildpackRootBlob constructs a buildpack from a blob. It is assumed that the buildpack contents reside at the
// root of the blob. The constructed buildpack contents will be structured as per the distribution spec (currently
// a tar with contents under '/cnb/buildpacks/{ID}/{version}/*').
func FromBuildpackRootBlob(blob Blob, layerWriterFactory archive.TarWriterFactory) (BuildModule, error) {
	descriptor := dist.BuildpackDescriptor{}
	descriptor.WithAPI = api.MustParse(dist.AssumedBuildpackAPIVersion)
	if err := readDescriptor(KindBuildpack, &descriptor, blob); err != nil {
		return nil, err
	}
	if err := detectPlatformSpecificValues(&descriptor, blob); err != nil {
		return nil, err
	}
	if err := validateBuildpackDescriptor(descriptor); err != nil {
		return nil, err
	}
	return buildpackFrom(&descriptor, blob, layerWriterFactory)
}

// FromExtensionRootBlob constructs an extension from a blob. It is assumed that the extension contents reside at the
// root of the blob. The constructed extension contents will be structured as per the distribution spec (currently
// a tar with contents under '/cnb/extensions/{ID}/{version}/*').
func FromExtensionRootBlob(blob Blob, layerWriterFactory archive.TarWriterFactory) (BuildModule, error) {
	descriptor := dist.ExtensionDescriptor{}
	descriptor.WithAPI = api.MustParse(dist.AssumedBuildpackAPIVersion)
	if err := readDescriptor(KindExtension, &descriptor, blob); err != nil {
		return nil, err
	}
	if err := validateExtensionDescriptor(descriptor); err != nil {
		return nil, err
	}
	return buildpackFrom(&descriptor, blob, layerWriterFactory)
}

func readDescriptor(kind string, descriptor interface{}, blob Blob) error {
	rc, err := blob.Open()
	if err != nil {
		return errors.Wrapf(err, "open %s", kind)
	}
	defer rc.Close()

	descriptorFile := kind + ".toml"

	_, buf, err := archive.ReadTarEntry(rc, descriptorFile)
	if err != nil {
		return errors.Wrapf(err, "reading %s", descriptorFile)
	}

	_, err = toml.Decode(string(buf), descriptor)
	if err != nil {
		return errors.Wrapf(err, "decoding %s", descriptorFile)
	}

	return nil
}

func detectPlatformSpecificValues(descriptor *dist.BuildpackDescriptor, blob Blob) error {
	if val, err := hasFile(blob, path.Join("bin", "build")); val {
		descriptor.WithLinuxBuild = true
	} else if err != nil {
		return err
	}
	if val, err := hasFile(blob, path.Join("bin", "build.bat")); val {
		descriptor.WithWindowsBuild = true
	} else if err != nil {
		return err
	}
	if val, err := hasFile(blob, path.Join("bin", "build.exe")); val {
		descriptor.WithWindowsBuild = true
	} else if err != nil {
		return err
	}
	return nil
}

func hasFile(blob Blob, file string) (bool, error) {
	rc, err := blob.Open()
	if err != nil {
		return false, errors.Wrapf(err, "open %s", "buildpack bin/")
	}
	defer rc.Close()
	_, _, err = archive.ReadTarEntry(rc, file)
	return err == nil, nil
}

func buildpackFrom(descriptor Descriptor, blob Blob, layerWriterFactory archive.TarWriterFactory) (BuildModule, error) {
	return &buildModule{
		descriptor: descriptor,
		Blob: &distBlob{
			openFn: func() io.ReadCloser {
				return archive.GenerateTarWithWriter(
					func(tw archive.TarWriter) error {
						return toDistTar(tw, descriptor, blob)
					},
					layerWriterFactory,
				)
			},
		},
	}, nil
}

type distBlob struct {
	openFn func() io.ReadCloser
}

func (b *distBlob) Open() (io.ReadCloser, error) {
	return b.openFn(), nil
}

func toDistTar(tw archive.TarWriter, descriptor Descriptor, blob Blob) error {
	ts := archive.NormalizedDateTime

	parentDir := dist.BuildpacksDir
	if descriptor.Kind() == KindExtension {
		parentDir = dist.ExtensionsDir
	}

	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path.Join(parentDir, descriptor.EscapedID()),
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return errors.Wrapf(err, "writing %s id dir header", descriptor.Kind())
	}

	baseTarDir := path.Join(parentDir, descriptor.EscapedID(), descriptor.Info().Version)
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     baseTarDir,
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return errors.Wrapf(err, "writing %s version dir header", descriptor.Kind())
	}

	rc, err := blob.Open()
	if err != nil {
		return errors.Wrapf(err, "reading %s blob", descriptor.Kind())
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get next tar entry")
		}

		archive.NormalizeHeader(header, true)
		header.Name = path.Clean(header.Name)
		if header.Name == "." || header.Name == "/" {
			continue
		}

		header.Mode = calcFileMode(header)
		header.Name = path.Join(baseTarDir, header.Name)
		err = tw.WriteHeader(header)
		if err != nil {
			return errors.Wrapf(err, "failed to write header for '%s'", header.Name)
		}

		_, err = io.Copy(tw, tr)
		if err != nil {
			return errors.Wrapf(err, "failed to write contents to '%s'", header.Name)
		}
	}

	return nil
}

func calcFileMode(header *tar.Header) int64 {
	switch {
	case header.Typeflag == tar.TypeDir:
		return 0755
	case nameOneOf(header.Name,
		path.Join("bin", "build"),
		path.Join("bin", "detect"),
		path.Join("bin", "generate"),
	):
		return 0755
	case anyExecBit(header.Mode):
		return 0755
	}

	return 0644
}

func nameOneOf(name string, paths ...string) bool {
	for _, p := range paths {
		if name == p {
			return true
		}
	}
	return false
}

func anyExecBit(mode int64) bool {
	return mode&0111 != 0
}

func validateBuildpackDescriptor(bpd dist.BuildpackDescriptor) error {
	if bpd.Info().ID == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.id"))
	}

	if bpd.Info().Version == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.version"))
	}

	if len(bpd.Order()) >= 1 && (len(bpd.Stacks()) >= 1 || len(bpd.Targets()) >= 1) {
		return errors.Errorf(
			"buildpack %s: cannot have both %s/%s and an %s defined",
			style.Symbol(bpd.Info().FullName()),
			style.Symbol("targets"),
			style.Symbol("stacks"),
			style.Symbol("order"),
		)
	}

	return nil
}

func validateExtensionDescriptor(extd dist.ExtensionDescriptor) error {
	if extd.Info().ID == "" {
		return errors.Errorf("%s is required", style.Symbol("extension.id"))
	}

	if extd.Info().Version == "" {
		return errors.Errorf("%s is required", style.Symbol("extension.version"))
	}

	return nil
}

func ToLayerTar(dest string, module BuildModule) (string, error) {
	descriptor := module.Descriptor()
	modReader, err := module.Open()
	if err != nil {
		return "", errors.Wrap(err, "opening blob")
	}
	defer modReader.Close()

	layerTar := filepath.Join(dest, fmt.Sprintf("%s.%s.tar", descriptor.EscapedID(), descriptor.Info().Version))
	fh, err := os.Create(layerTar)
	if err != nil {
		return "", errors.Wrap(err, "create file for tar")
	}
	defer fh.Close()

	if _, err := io.Copy(fh, modReader); err != nil {
		return "", errors.Wrap(err, "writing blob to tar")
	}

	return layerTar, nil
}

func ToNLayerTar(dest string, module BuildModule) ([]ModuleTar, error) {
	modReader, err := module.Open()
	if err != nil {
		return nil, errors.Wrap(err, "opening blob")
	}
	defer modReader.Close()

	tarCollection := newModuleTarCollection(dest)
	tr := tar.NewReader(modReader)

	var (
		header     *tar.Header
		forWindows bool
	)

	for {
		header, err = tr.Next()
		if err != nil {
			if err == io.EOF {
				return handleEmptyModule(dest, module)
			}
			return nil, err
		}
		if _, err := sanitizePath(header.Name); err != nil {
			return nil, err
		}
		if header.Name == "Files" {
			forWindows = true
		}
		if strings.Contains(header.Name, `/cnb/buildpacks/`) || strings.Contains(header.Name, `\cnb\buildpacks\`) {
			// Only for Windows, the first four headers are:
			// - Files
			// - Hives
			// - Files/cnb
			// - Files/cnb/buildpacks
			// Skip over these until we find "Files/cnb/buildpacks/<buildpack-id>":
			break
		}
	}
	// The header should look like "/cnb/buildpacks/<buildpack-id>"
	// The version should be blank because the first header is missing <buildpack-version>.
	origID, origVersion := parseBpIDAndVersion(header)
	if origVersion != "" {
		return nil, fmt.Errorf("first header '%s' contained unexpected version", header.Name)
	}

	if err := toNLayerTar(origID, origVersion, header, tr, tarCollection, forWindows); err != nil {
		return nil, err
	}

	errs := tarCollection.close()
	if len(errs) > 0 {
		return nil, errors.New("closing files")
	}

	return tarCollection.moduleTars(), nil
}

func toNLayerTar(origID, origVersion string, firstHeader *tar.Header, tr *tar.Reader, tc *moduleTarCollection, forWindows bool) error {
	toWrite := []*tar.Header{firstHeader}
	if origVersion == "" {
		// the first header only contains the id - e.g., /cnb/buildpacks/<buildpack-id>,
		// read the next header to get the version
		secondHeader, err := tr.Next()
		if err != nil {
			return fmt.Errorf("getting second header: %w; first header was %s", err, firstHeader.Name)
		}
		if _, err := sanitizePath(secondHeader.Name); err != nil {
			return err
		}
		nextID, nextVersion := parseBpIDAndVersion(secondHeader)
		if nextID != origID || nextVersion == "" {
			return fmt.Errorf("second header '%s' contained unexpected id or missing version", secondHeader.Name)
		}
		origVersion = nextVersion
		toWrite = append(toWrite, secondHeader)
	} else {
		// the first header contains id and version - e.g., /cnb/buildpacks/<buildpack-id>/<buildpack-version>,
		// we need to write the parent header - e.g., /cnb/buildpacks/<buildpack-id>
		realFirstHeader := *firstHeader
		realFirstHeader.Name = filepath.ToSlash(filepath.Dir(firstHeader.Name))
		toWrite = append([]*tar.Header{&realFirstHeader}, toWrite...)
	}
	if forWindows {
		toWrite = append(windowsPreamble(), toWrite...)
	}
	mt, err := tc.get(origID, origVersion)
	if err != nil {
		return fmt.Errorf("getting module from collection: %w", err)
	}
	for _, h := range toWrite {
		if err := mt.writer.WriteHeader(h); err != nil {
			return fmt.Errorf("failed to write header '%s': %w", h.Name, err)
		}
	}
	// write the rest of the package
	var header *tar.Header
	for {
		header, err = tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("getting next header: %w", err)
		}
		if _, err := sanitizePath(header.Name); err != nil {
			return err
		}
		nextID, nextVersion := parseBpIDAndVersion(header)
		if nextID != origID || nextVersion != origVersion {
			// we found a new module, recurse
			return toNLayerTar(nextID, nextVersion, header, tr, tc, forWindows)
		}

		err = mt.writer.WriteHeader(header)
		if err != nil {
			return fmt.Errorf("failed to write header for '%s': %w", header.Name, err)
		}

		buf, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("failed to read contents of '%s': %w", header.Name, err)
		}

		_, err = mt.writer.Write(buf)
		if err != nil {
			return fmt.Errorf("failed to write contents to '%s': %w", header.Name, err)
		}
	}
}

func sanitizePath(path string) (string, error) {
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path %s contains unexpected special elements", path)
	}
	return path, nil
}

func windowsPreamble() []*tar.Header {
	return []*tar.Header{
		{
			Name:     "Files",
			Typeflag: tar.TypeDir,
		},
		{
			Name:     "Hives",
			Typeflag: tar.TypeDir,
		},
		{
			Name:     "Files/cnb",
			Typeflag: tar.TypeDir,
		},
		{
			Name:     "Files/cnb/buildpacks",
			Typeflag: tar.TypeDir,
		},
	}
}

func parseBpIDAndVersion(hdr *tar.Header) (id, version string) {
	// splitting "/cnb/buildpacks/{ID}/{version}/*" returns
	// [0] = "" -> first element is empty or "Files" in windows
	// [1] = "cnb"
	// [2] = "buildpacks"
	// [3] = "{ID}"
	// [4] = "{version}"
	// ...
	parts := strings.Split(strings.ReplaceAll(filepath.Clean(hdr.Name), `\`, `/`), `/`)
	size := len(parts)
	switch {
	case size < 4:
		// error
	case size == 4:
		id = parts[3]
	case size >= 5:
		id = parts[3]
		version = parts[4]
	}
	return id, version
}

func handleEmptyModule(dest string, module BuildModule) ([]ModuleTar, error) {
	tarFile, err := ToLayerTar(dest, module)
	if err != nil {
		return nil, err
	}
	layerTar := &moduleTar{
		info: module.Descriptor().Info(),
		path: tarFile,
	}
	return []ModuleTar{layerTar}, nil
}

// Set returns a set of the given string slice.
func Set(exclude []string) map[string]struct{} {
	type void struct{}
	var member void
	var excludedModules = make(map[string]struct{})
	for _, fullName := range exclude {
		excludedModules[fullName] = member
	}
	return excludedModules
}

type ModuleTar interface {
	Info() dist.ModuleInfo
	Path() string
}

type moduleTar struct {
	info   dist.ModuleInfo
	path   string
	writer archive.TarWriter
}

func (t *moduleTar) Info() dist.ModuleInfo {
	return t.info
}

func (t *moduleTar) Path() string {
	return t.path
}

func newModuleTar(dest, id, version string) (moduleTar, error) {
	layerTar := filepath.Join(dest, fmt.Sprintf("%s.%s.tar", id, version))
	fh, err := os.Create(layerTar)
	if err != nil {
		return moduleTar{}, errors.Wrapf(err, "creating file at path %s", layerTar)
	}
	return moduleTar{
		info: dist.ModuleInfo{
			ID:      id,
			Version: version,
		},
		path:   layerTar,
		writer: tar.NewWriter(fh),
	}, nil
}

type moduleTarCollection struct {
	rootPath string
	modules  map[string]moduleTar
}

func newModuleTarCollection(rootPath string) *moduleTarCollection {
	return &moduleTarCollection{
		rootPath: rootPath,
		modules:  map[string]moduleTar{},
	}
}

func (m *moduleTarCollection) get(id, version string) (moduleTar, error) {
	key := fmt.Sprintf("%s@%s", id, version)
	if _, ok := m.modules[key]; !ok {
		module, err := newModuleTar(m.rootPath, id, version)
		if err != nil {
			return moduleTar{}, err
		}
		m.modules[key] = module
	}
	return m.modules[key], nil
}

func (m *moduleTarCollection) moduleTars() []ModuleTar {
	var modulesTar []ModuleTar
	for _, v := range m.modules {
		v := v
		vv := &v
		modulesTar = append(modulesTar, vv)
	}
	return modulesTar
}

func (m *moduleTarCollection) close() []error {
	var errors []error
	for _, v := range m.modules {
		err := v.writer.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
