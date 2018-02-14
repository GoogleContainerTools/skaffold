package docker

import (
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func CreateDockerTarContext(w io.Writer, dockerfilePath, context string) error {
	f, err := os.Open(dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "opening dockerfile")
	}
	paths, err := GetDockerfileDependencies(context, f)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}
	f.Close()
	absDockerfilePath, _ := filepath.Abs(dockerfilePath)

	paths = append(paths, absDockerfilePath)
	if err := util.CreateTarGz(w, context, paths); err != nil {
		return errors.Wrap(err, "creating tar gz")
	}
	return nil
}
