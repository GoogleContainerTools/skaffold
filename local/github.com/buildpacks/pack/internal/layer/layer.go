package layer

import (
	"github.com/buildpacks/pack/pkg/archive"
)

func CreateSingleFileTar(tarFile, path, txt string, twf archive.TarWriterFactory) error {
	tarBuilder := archive.TarBuilder{}
	tarBuilder.AddFile(path, 0644, archive.NormalizedDateTime, []byte(txt))
	return tarBuilder.WriteToPath(tarFile, twf)
}
