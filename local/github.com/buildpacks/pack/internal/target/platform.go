package target

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

func getPlatform(t []string, logger logging.Logger) (os, arch, variant string, err error) {
	os, _ = getSliceAt[string](t, 0)
	arch, _ = getSliceAt[string](t, 1)
	variant, _ = getSliceAt[string](t, 2)
	if !supportsOS(os) && supportsVariant(arch, variant) {
		logger.Warn(style.Warn("unknown os %s, is this a typo", os))
	}
	if supportsArch(os, arch) && !supportsVariant(arch, variant) {
		logger.Warn(style.Warn("unknown variant %s", variant))
	}
	if supportsOS(os) && !supportsArch(os, arch) && supportsVariant(arch, variant) {
		logger.Warn(style.Warn("unknown arch %s", arch))
	}
	if !SupportsPlatform(os, arch, variant) {
		return os, arch, variant, errors.Errorf("unknown target: %s", style.Symbol(strings.Join(t, "/")))
	}
	return os, arch, variant, err
}

var supportedOSArchs = map[string][]string{
	"aix":       {"ppc64"},
	"android":   {"386", "amd64", "arm", "arm64"},
	"darwin":    {"amd64", "arm64"},
	"dragonfly": {"amd64"},
	"freebsd":   {"386", "amd64", "arm"},
	"illumos":   {"amd64"},
	"ios":       {"arm64"},
	"js":        {"wasm"},
	"linux":     {"386", "amd64", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le", "riscv64", "s390x"},
	"netbsd":    {"386", "amd64", "arm"},
	"openbsd":   {"386", "amd64", "arm", "arm64"},
	"plan9":     {"386", "amd64", "arm"},
	"solaris":   {"amd64"},
	"wasip1":    {"wasm"},
	"windows":   {"386", "amd64", "arm", "arm64"},
}

var supportedArchVariants = map[string][]string{
	"386":      {"softfloat", "sse2"},
	"arm":      {"v5", "v6", "v7"},
	"amd64":    {"v1", "v2", "v3", "v4"},
	"mips":     {"hardfloat", "softfloat"},
	"mipsle":   {"hardfloat", "softfloat"},
	"mips64":   {"hardfloat", "softfloat"},
	"mips64le": {"hardfloat", "softfloat"},
	"ppc64":    {"power8", "power9"},
	"ppc64le":  {"power8", "power9"},
	"wasm":     {"satconv", "signext"},
}

func supportsOS(os string) bool {
	return supportedOSArchs[os] != nil
}

func supportsArch(os, arch string) bool {
	if supportsOS(os) {
		for _, s := range supportedOSArchs[os] {
			if s == arch {
				return true
			}
		}
	}
	return false
}

func supportsVariant(arch, variant string) (supported bool) {
	if variant == "" || len(variant) == 0 {
		return true
	}
	for _, s := range supportedArchVariants[arch] {
		if s == variant {
			return true
		}
	}
	return supported
}

func SupportsPlatform(os, arch, variant string) bool {
	return supportsArch(os, arch) && supportsVariant(arch, variant)
}
