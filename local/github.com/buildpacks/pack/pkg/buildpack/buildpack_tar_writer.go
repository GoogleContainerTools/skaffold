package buildpack

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/logging"
)

type BuildModuleWriter struct {
	logger  logging.Logger
	factory archive.TarWriterFactory
}

// NewBuildModuleWriter creates a BuildModule writer
func NewBuildModuleWriter(logger logging.Logger, factory archive.TarWriterFactory) *BuildModuleWriter {
	return &BuildModuleWriter{
		logger:  logger,
		factory: factory,
	}
}

// NToLayerTar creates a tar file containing the all the Buildpacks given, but excluding the ones which FullName() is
// in the exclude list. It returns the path to the tar file, the list of Buildpacks that were excluded, and any error
func (b *BuildModuleWriter) NToLayerTar(tarPath, filename string, modules []BuildModule, exclude map[string]struct{}) (string, []BuildModule, error) {
	layerTar := filepath.Join(tarPath, fmt.Sprintf("%s.tar", filename))
	tarFile, err := os.Create(layerTar)
	b.logger.Debugf("creating file %s", style.Symbol(layerTar))
	if err != nil {
		return "", nil, errors.Wrap(err, "create file for tar")
	}

	defer tarFile.Close()
	tw := b.factory.NewWriter(tarFile)
	defer tw.Close()

	parentFolderAdded := map[string]bool{}
	duplicated := map[string]bool{}

	var buildModuleExcluded []BuildModule
	for _, module := range modules {
		if _, ok := exclude[module.Descriptor().Info().FullName()]; !ok {
			if !duplicated[module.Descriptor().Info().FullName()] {
				duplicated[module.Descriptor().Info().FullName()] = true
				b.logger.Debugf("adding %s", style.Symbol(module.Descriptor().Info().FullName()))

				if err := b.writeBuildModuleToTar(tw, module, &parentFolderAdded); err != nil {
					return "", nil, errors.Wrapf(err, "adding %s", style.Symbol(module.Descriptor().Info().FullName()))
				}
				rootPath := processRootPath(module)
				if !parentFolderAdded[rootPath] {
					parentFolderAdded[rootPath] = true
				}
			} else {
				b.logger.Debugf("skipping %s, it was already added", style.Symbol(module.Descriptor().Info().FullName()))
			}
		} else {
			b.logger.Debugf("excluding %s from being flattened", style.Symbol(module.Descriptor().Info().FullName()))
			buildModuleExcluded = append(buildModuleExcluded, module)
		}
	}

	b.logger.Debugf("%s was created successfully", style.Symbol(layerTar))
	return layerTar, buildModuleExcluded, nil
}

// writeBuildModuleToTar writes the content of the given tar file into the writer, skipping the folders that were already added
func (b *BuildModuleWriter) writeBuildModuleToTar(tw archive.TarWriter, module BuildModule, parentFolderAdded *map[string]bool) error {
	var (
		rc  io.ReadCloser
		err error
	)

	if rc, err = module.Open(); err != nil {
		return err
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

		if (*parentFolderAdded)[header.Name] {
			b.logger.Debugf("folder %s was already added, skipping it", style.Symbol(header.Name))
			continue
		}

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

func processRootPath(module BuildModule) string {
	var bpFolder string
	switch module.Descriptor().Kind() {
	case buildpack.KindBuildpack:
		bpFolder = "buildpacks"
	case buildpack.KindExtension:
		bpFolder = "extensions"
	default:
		bpFolder = "buildpacks"
	}
	bpInfo := module.Descriptor().Info()
	rootPath := path.Join("/cnb", bpFolder, strings.ReplaceAll(bpInfo.ID, "/", "_"))
	return rootPath
}
