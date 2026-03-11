// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package loader

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"

	"dario.cat/mergo"
	"github.com/docker/cli/cli/compose/types"
)

type specials struct {
	m map[reflect.Type]func(dst, src reflect.Value) error
}

func (s *specials) Transformer(t reflect.Type) func(dst, src reflect.Value) error {
	if fn, ok := s.m[t]; ok {
		return fn
	}
	return nil
}

func merge(configs []*types.Config) (*types.Config, error) {
	base := configs[0]
	for _, override := range configs[1:] {
		var errs []error
		if services, err := mergeServices(base.Services, override.Services); err != nil {
			errs = append(errs, fmt.Errorf("cannot merge services: %w", err))
		} else {
			base.Services = services
		}
		if err := mergo.Map(&base.Volumes, &override.Volumes, mergo.WithOverride); err != nil {
			errs = append(errs, fmt.Errorf("cannot merge volumes: %w", err))
		}
		if err := mergo.Map(&base.Networks, &override.Networks, mergo.WithOverride); err != nil {
			errs = append(errs, fmt.Errorf("cannot merge networks: %w", err))
		}
		if err := mergo.Map(&base.Secrets, &override.Secrets, mergo.WithOverride); err != nil {
			errs = append(errs, fmt.Errorf("cannot merge secrets: %w", err))
		}
		if err := mergo.Map(&base.Configs, &override.Configs, mergo.WithOverride); err != nil {
			errs = append(errs, fmt.Errorf("cannot merge configs: %w", err))
		}
		if err := errors.Join(errs...); err != nil {
			return nil, errors.Join(fmt.Errorf("failed to merge file %s", override.Filename), err)
		}
	}
	return base, nil
}

func mergeServices(base, override []types.ServiceConfig) ([]types.ServiceConfig, error) {
	mergeOpts := []func(*mergo.Config){
		mergo.WithAppendSlice,
		mergo.WithOverride,
		mergo.WithTransformers(&specials{m: map[reflect.Type]func(dst, src reflect.Value) error{
			reflect.PointerTo(reflect.TypeFor[types.LoggingConfig]()):        safelyMerge(mergeLoggingConfig),
			reflect.TypeFor[[]types.ServicePortConfig]():                     mergeSlice(toServicePortConfigsMap, toServicePortConfigsSlice),
			reflect.TypeFor[[]types.ServiceSecretConfig]():                   mergeSlice(toServiceSecretConfigsMap, toServiceSecretConfigsSlice),
			reflect.TypeFor[[]types.ServiceConfigObjConfig]():                mergeSlice(toServiceConfigObjConfigsMap, toSServiceConfigObjConfigsSlice),
			reflect.PointerTo(reflect.TypeFor[types.UlimitsConfig]()):        mergeUlimitsConfig,
			reflect.TypeFor[[]types.ServiceVolumeConfig]():                   mergeSlice(toServiceVolumeConfigsMap, toServiceVolumeConfigsSlice),
			reflect.TypeFor[types.ShellCommand]():                            mergeShellCommand,
			reflect.PointerTo(reflect.TypeFor[types.ServiceNetworkConfig]()): mergeServiceNetworkConfig,
			reflect.PointerTo(reflect.TypeFor[uint64]()):                     mergeUint64,
		}}),
	}

	baseServices := make(map[string]types.ServiceConfig, len(base))
	for _, s := range base {
		baseServices[s.Name] = s
	}

	for _, overrideService := range override {
		if baseService, ok := baseServices[overrideService.Name]; ok {
			if err := mergo.Merge(&baseService, &overrideService, mergeOpts...); err != nil {
				return nil, fmt.Errorf("cannot merge service %s: %w", overrideService.Name, err)
			}
			baseServices[overrideService.Name] = baseService
			continue
		}
		baseServices[overrideService.Name] = overrideService
	}

	services := make([]types.ServiceConfig, 0, len(baseServices))
	for _, baseService := range baseServices {
		services = append(services, baseService)
	}

	slices.SortFunc(services, func(a, b types.ServiceConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return services, nil
}

func toServiceSecretConfigsMap(s any) (map[any]any, error) {
	secrets, ok := s.([]types.ServiceSecretConfig)
	if !ok {
		return nil, fmt.Errorf("not a serviceSecretConfig: %v", s)
	}
	m := map[any]any{}
	for _, secret := range secrets {
		m[secret.Source] = secret
	}
	return m, nil
}

func toServiceConfigObjConfigsMap(s any) (map[any]any, error) {
	secrets, ok := s.([]types.ServiceConfigObjConfig)
	if !ok {
		return nil, fmt.Errorf("not a serviceSecretConfig: %v", s)
	}
	m := map[any]any{}
	for _, secret := range secrets {
		m[secret.Source] = secret
	}
	return m, nil
}

func toServicePortConfigsMap(s any) (map[any]any, error) {
	ports, ok := s.([]types.ServicePortConfig)
	if !ok {
		return nil, fmt.Errorf("not a servicePortConfig slice: %v", s)
	}
	m := map[any]any{}
	for _, p := range ports {
		m[p.Published] = p
	}
	return m, nil
}

func toServiceVolumeConfigsMap(s any) (map[any]any, error) {
	volumes, ok := s.([]types.ServiceVolumeConfig)
	if !ok {
		return nil, fmt.Errorf("not a serviceVolumeConfig slice: %v", s)
	}
	m := map[any]any{}
	for _, v := range volumes {
		m[v.Target] = v
	}
	return m, nil
}

func toServiceSecretConfigsSlice(dst reflect.Value, m map[any]any) error {
	s := make([]types.ServiceSecretConfig, 0, len(m))
	for _, v := range m {
		s = append(s, v.(types.ServiceSecretConfig))
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Source < s[j].Source })
	dst.Set(reflect.ValueOf(s))
	return nil
}

func toSServiceConfigObjConfigsSlice(dst reflect.Value, m map[any]any) error {
	s := make([]types.ServiceConfigObjConfig, 0, len(m))
	for _, v := range m {
		s = append(s, v.(types.ServiceConfigObjConfig))
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Source < s[j].Source })
	dst.Set(reflect.ValueOf(s))
	return nil
}

func toServicePortConfigsSlice(dst reflect.Value, m map[any]any) error {
	s := make([]types.ServicePortConfig, 0, len(m))
	for _, v := range m {
		s = append(s, v.(types.ServicePortConfig))
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Published < s[j].Published })
	dst.Set(reflect.ValueOf(s))
	return nil
}

func toServiceVolumeConfigsSlice(dst reflect.Value, m map[any]any) error {
	s := make([]types.ServiceVolumeConfig, 0, len(m))
	for _, v := range m {
		s = append(s, v.(types.ServiceVolumeConfig))
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Target < s[j].Target })
	dst.Set(reflect.ValueOf(s))
	return nil
}

type (
	tomapFn             func(s any) (map[any]any, error)
	writeValueFromMapFn func(reflect.Value, map[any]any) error
)

func safelyMerge(mergeFn func(dst, src reflect.Value) error) func(dst, src reflect.Value) error {
	return func(dst, src reflect.Value) error {
		if src.IsNil() {
			return nil
		}
		if dst.IsNil() {
			dst.Set(src)
			return nil
		}
		return mergeFn(dst, src)
	}
}

func mergeSlice(tomap tomapFn, writeValue writeValueFromMapFn) func(dst, src reflect.Value) error {
	return func(dst, src reflect.Value) error {
		dstMap, err := sliceToMap(tomap, dst)
		if err != nil {
			return err
		}
		srcMap, err := sliceToMap(tomap, src)
		if err != nil {
			return err
		}
		if err := mergo.Map(&dstMap, srcMap, mergo.WithOverride); err != nil {
			return err
		}
		return writeValue(dst, dstMap)
	}
}

func sliceToMap(tomap tomapFn, v reflect.Value) (map[any]any, error) {
	// check if valid
	if !v.IsValid() {
		return nil, fmt.Errorf("invalid value : %+v", v)
	}
	return tomap(v.Interface())
}

func mergeLoggingConfig(dst, src reflect.Value) error {
	dstDriver := dst.Elem().FieldByName("Driver").String()
	srcDriver := src.Elem().FieldByName("Driver").String()

	// Same driver, merging options
	if dstDriver == srcDriver || dstDriver == "" || srcDriver == "" {
		if dstDriver == "" {
			dst.Elem().FieldByName("Driver").SetString(srcDriver)
		}
		dstOptions := dst.Elem().FieldByName("Options").Interface().(map[string]string)
		srcOptions := src.Elem().FieldByName("Options").Interface().(map[string]string)
		return mergo.Merge(&dstOptions, srcOptions, mergo.WithOverride)
	}
	// Different driver, override with src
	dst.Set(src)
	return nil
}

//nolint:unparam
func mergeUlimitsConfig(dst, src reflect.Value) error {
	if src.Interface() != reflect.Zero(reflect.TypeOf(src.Interface())).Interface() {
		dst.Elem().Set(src.Elem())
	}
	return nil
}

//nolint:unparam
func mergeShellCommand(dst, src reflect.Value) error {
	if src.Len() != 0 {
		dst.Set(src)
	}
	return nil
}

//nolint:unparam
func mergeServiceNetworkConfig(dst, src reflect.Value) error {
	if src.Interface() != reflect.Zero(reflect.TypeOf(src.Interface())).Interface() {
		dst.Elem().FieldByName("Aliases").Set(src.Elem().FieldByName("Aliases"))
		if ipv4 := src.Elem().FieldByName("Ipv4Address").Interface().(string); ipv4 != "" {
			dst.Elem().FieldByName("Ipv4Address").SetString(ipv4)
		}
		if ipv6 := src.Elem().FieldByName("Ipv6Address").Interface().(string); ipv6 != "" {
			dst.Elem().FieldByName("Ipv6Address").SetString(ipv6)
		}
	}
	return nil
}

//nolint:unparam
func mergeUint64(dst, src reflect.Value) error {
	if !src.IsNil() {
		dst.Elem().Set(src.Elem())
	}
	return nil
}
