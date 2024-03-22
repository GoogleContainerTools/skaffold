package name

import (
	"fmt"
	"strings"

	gname "github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/style"
)

const (
	defaultRefFormat = "%s/%s:%s"
	digestRefFormat  = "%s/%s@%s"
)

type Logger interface {
	Infof(fmt string, v ...interface{})
}

func TranslateRegistry(name string, registryMirrors map[string]string, logger Logger) (string, error) {
	if registryMirrors == nil {
		return name, nil
	}

	srcRef, err := gname.ParseReference(name, gname.WeakValidation)
	if err != nil {
		return "", err
	}

	srcContext := srcRef.Context()
	registryMirror, ok := getMirror(srcContext, registryMirrors)
	if !ok {
		return name, nil
	}

	refFormat := defaultRefFormat
	if strings.Contains(srcRef.Identifier(), ":") {
		refFormat = digestRefFormat
	}

	refName := fmt.Sprintf(refFormat, registryMirror, srcContext.RepositoryStr(), srcRef.Identifier())
	_, err = gname.ParseReference(refName, gname.WeakValidation)
	if err != nil {
		return "", err
	}

	logger.Infof("Using mirror %s for %s", style.Symbol(refName), name)
	return refName, nil
}

func getMirror(repo gname.Repository, registryMirrors map[string]string) (string, bool) {
	mirror, ok := registryMirrors["*"]
	if ok {
		return mirror, ok
	}

	mirror, ok = registryMirrors[repo.RegistryStr()]
	return mirror, ok
}
