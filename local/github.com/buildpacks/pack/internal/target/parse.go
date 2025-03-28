package target

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

func ParseTargets(t []string, logger logging.Logger) (targets []dist.Target, err error) {
	for _, v := range t {
		target, err := ParseTarget(v, logger)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func ParseTarget(t string, logger logging.Logger) (output dist.Target, err error) {
	nonDistro, distros, err := getTarget(t, logger)
	if v, _ := getSliceAt[string](nonDistro, 0); len(nonDistro) <= 1 && v == "" {
		logger.Warn("os/arch must be defined")
	}
	if err != nil {
		return output, err
	}
	os, arch, variant, err := getPlatform(nonDistro, logger)
	if err != nil {
		return output, err
	}
	v, err := ParseDistros(distros, logger)
	if err != nil {
		return output, err
	}
	output = dist.Target{
		OS:            os,
		Arch:          arch,
		ArchVariant:   variant,
		Distributions: v,
	}
	return output, err
}

func ParseDistros(distroSlice string, logger logging.Logger) (distros []dist.Distribution, err error) {
	distro := strings.Split(distroSlice, ";")
	if l := len(distro); l == 1 && distro[0] == "" {
		return nil, err
	}
	for _, d := range distro {
		v, err := ParseDistro(d, logger)
		if err != nil {
			return nil, err
		}
		distros = append(distros, v)
	}
	return distros, nil
}

func ParseDistro(distroString string, logger logging.Logger) (distro dist.Distribution, err error) {
	d := strings.Split(distroString, "@")
	if d[0] == "" || len(d) == 0 {
		return distro, errors.Errorf("distro's versions %s cannot be specified without distro's name", style.Symbol("@"+strings.Join(d[1:], "@")))
	}
	distro.Name = d[0]
	if len(d) < 2 {
		logger.Warnf("distro with name %s has no specific version!", style.Symbol(d[0]))
		return distro, err
	}
	if len(d) > 2 {
		return distro, fmt.Errorf("invalid distro: %s", distroString)
	}
	distro.Version = d[1]
	return distro, err
}

func getTarget(t string, logger logging.Logger) (nonDistro []string, distros string, err error) {
	target := strings.Split(t, ":")
	if (len(target) == 1 && target[0] == "") || len(target) == 0 {
		return nonDistro, distros, errors.Errorf("invalid target %s, atleast one of [os][/arch][/archVariant] must be specified", t)
	}
	if len(target) == 2 && target[0] == "" {
		v, _ := getSliceAt[string](target, 1)
		logger.Warn(style.Warn("adding distros %s without [os][/arch][/variant]", v))
	} else {
		i, _ := getSliceAt[string](target, 0)
		nonDistro = strings.Split(i, "/")
	}
	if i, err := getSliceAt[string](target, 1); err == nil {
		distros = i
	}
	return nonDistro, distros, err
}

func getSliceAt[T interface{}](slice []T, index int) (value T, err error) {
	if index < 0 || index >= len(slice) {
		return value, errors.Errorf("index out of bound, cannot access item at index %d of slice with length %d", index, len(slice))
	}

	return slice[index], err
}
