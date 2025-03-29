package stack

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
)

const MixinsLabel = "io.buildpacks.stack.mixins"

func ValidateMixins(buildImageName string, buildImageMixins []string, runImageName string, runImageMixins []string) error {
	if invalid := FindStageMixins(buildImageMixins, "run"); len(invalid) > 0 {
		sort.Strings(invalid)
		return fmt.Errorf("%s contains run-only mixin(s): %s", style.Symbol(buildImageName), strings.Join(invalid, ", "))
	}

	if invalid := FindStageMixins(runImageMixins, "build"); len(invalid) > 0 {
		sort.Strings(invalid)
		return fmt.Errorf("%s contains build-only mixin(s): %s", style.Symbol(runImageName), strings.Join(invalid, ", "))
	}

	buildImageMixins = removeStageMixins(buildImageMixins, "build")
	runImageMixins = removeStageMixins(runImageMixins, "run")

	_, missing, _ := stringset.Compare(runImageMixins, buildImageMixins)

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s missing required mixin(s): %s", style.Symbol(runImageName), strings.Join(missing, ", "))
	}
	return nil
}

func FindStageMixins(mixins []string, stage string) []string {
	var found []string
	for _, m := range mixins {
		if strings.HasPrefix(m, stage+":") {
			found = append(found, m)
		}
	}
	return found
}

func removeStageMixins(mixins []string, stage string) []string {
	var filtered []string
	for _, m := range mixins {
		if !strings.HasPrefix(m, stage+":") {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
