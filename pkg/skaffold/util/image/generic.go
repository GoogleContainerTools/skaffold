package image

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"regexp"
	"strings"
)

const maxLength = 255

const gcr = "gcr.io"
const escapeChars = "[/._:@]"
const prefixRegexStr = "gcr.io/[a-zA-Z0-9-_]+/"

var escapeRegex = regexp.MustCompile(escapeChars)
var prefixRegex = regexp.MustCompile(prefixRegexStr)

type GenericContainerRegistry struct {
	RegistryName string
}

func NewGenericContainerRegistry(name string) util.Registry{
	return &GenericContainerRegistry{name}
}

func (r *GenericContainerRegistry) String() string {
	return r.RegistryName
}

func (r *GenericContainerRegistry) Update(reg *util.Registry) util.Registry {
	return nil
}

func (r *GenericContainerRegistry) Prefix() string {
	return ""
}

func (r *GenericContainerRegistry) Postfix() string {
	return ""
}

type GenericImage struct {
	ImageRegistry util.Registry
	ImageName string
}

func NewGenericImage(reg util.Registry, name string) *GenericImage {
	return &GenericImage{reg, name}
}

func (i *GenericImage) Registry() util.Registry {
	return i.ImageRegistry
}

func (i *GenericImage) String() string {
	return i.ImageName
}

func (i *GenericImage) Update(reg util.Registry) string {
	originalImage := i.ImageRegistry.String() + "/" + i.String()
	if reg.String() == "" {
		return originalImage
	}
	if strings.HasPrefix(reg.String(), gcr) {
		originalPrefix := prefixRegex.FindString(originalImage)
		defaultRepoPrefix := prefixRegex.FindString(reg.String())

		if originalPrefix == defaultRepoPrefix {
			// prefixes match
			return reg.String() + "/" + originalImage[len(originalPrefix):]
		} else if strings.HasPrefix(originalImage, reg.String()) {
			return originalImage
		}
		// prefixes don't match, concatenate and truncate
		return truncate(reg.String() + "/" + originalImage)
	}
	return truncate(reg.String() + "/" + escapeRegex.ReplaceAllString(originalImage, "_"))
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
