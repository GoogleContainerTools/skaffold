package name

import (
	"fmt"
	"strings"

	"github.com/buildpacks/pack/pkg/dist"

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

func AppendSuffix(name string, target dist.Target) (string, error) {
	reference, err := gname.ParseReference(name, gname.WeakValidation)
	if err != nil {
		return "", err
	}

	suffixPlatformTag := targetToTag(target)
	if suffixPlatformTag != "" {
		if reference.Identifier() == "latest" {
			return fmt.Sprintf("%s:%s", reference.Context(), suffixPlatformTag), nil
		}
		if !strings.Contains(reference.Identifier(), ":") {
			return fmt.Sprintf("%s:%s-%s", reference.Context(), reference.Identifier(), suffixPlatformTag), nil
		}
	}
	return name, nil
}

func getMirror(repo gname.Repository, registryMirrors map[string]string) (string, bool) {
	mirror, ok := registryMirrors["*"]
	if ok {
		return mirror, ok
	}

	mirror, ok = registryMirrors[repo.RegistryStr()]
	return mirror, ok
}

func targetToTag(target dist.Target) string {
	return strings.Join(target.ValuesAsSlice(), "-")
}
