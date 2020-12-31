/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/apiversion"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha5"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta10"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta11"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta12"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta13"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta14"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta15"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta16"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta17"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta5"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta6"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta7"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta8"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta9"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2alpha4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta10"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta5"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta6"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta7"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta8"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta9"
	misc "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type APIVersion struct {
	Version string `yaml:"apiVersion"`
}

var SchemaVersions = Versions{
	{v1alpha1.Version, v1alpha1.NewSkaffoldConfig},
	{v1alpha2.Version, v1alpha2.NewSkaffoldConfig},
	{v1alpha3.Version, v1alpha3.NewSkaffoldConfig},
	{v1alpha4.Version, v1alpha4.NewSkaffoldConfig},
	{v1alpha5.Version, v1alpha5.NewSkaffoldConfig},
	{v1beta1.Version, v1beta1.NewSkaffoldConfig},
	{v1beta2.Version, v1beta2.NewSkaffoldConfig},
	{v1beta3.Version, v1beta3.NewSkaffoldConfig},
	{v1beta4.Version, v1beta4.NewSkaffoldConfig},
	{v1beta5.Version, v1beta5.NewSkaffoldConfig},
	{v1beta6.Version, v1beta6.NewSkaffoldConfig},
	{v1beta7.Version, v1beta7.NewSkaffoldConfig},
	{v1beta8.Version, v1beta8.NewSkaffoldConfig},
	{v1beta9.Version, v1beta9.NewSkaffoldConfig},
	{v1beta10.Version, v1beta10.NewSkaffoldConfig},
	{v1beta11.Version, v1beta11.NewSkaffoldConfig},
	{v1beta12.Version, v1beta12.NewSkaffoldConfig},
	{v1beta13.Version, v1beta13.NewSkaffoldConfig},
	{v1beta14.Version, v1beta14.NewSkaffoldConfig},
	{v1beta15.Version, v1beta15.NewSkaffoldConfig},
	{v1beta16.Version, v1beta16.NewSkaffoldConfig},
	{v1beta17.Version, v1beta17.NewSkaffoldConfig},
	{v1.Version, v1.NewSkaffoldConfig},
	{v2alpha1.Version, v2alpha1.NewSkaffoldConfig},
	{v2alpha2.Version, v2alpha2.NewSkaffoldConfig},
	{v2alpha3.Version, v2alpha3.NewSkaffoldConfig},
	{v2alpha4.Version, v2alpha4.NewSkaffoldConfig},
	{v2beta1.Version, v2beta1.NewSkaffoldConfig},
	{v2beta2.Version, v2beta2.NewSkaffoldConfig},
	{v2beta3.Version, v2beta3.NewSkaffoldConfig},
	{v2beta4.Version, v2beta4.NewSkaffoldConfig},
	{v2beta5.Version, v2beta5.NewSkaffoldConfig},
	{v2beta6.Version, v2beta6.NewSkaffoldConfig},
	{v2beta7.Version, v2beta7.NewSkaffoldConfig},
	{v2beta8.Version, v2beta8.NewSkaffoldConfig},
	{v2beta9.Version, v2beta9.NewSkaffoldConfig},
	{v2beta10.Version, v2beta10.NewSkaffoldConfig},
	{latest.Version, latest.NewSkaffoldConfig},
}

type Version struct {
	APIVersion string
	Factory    func() util.VersionedConfig
}

type Versions []Version

// Find search the constructor for a given api version.
func (v *Versions) Find(apiVersion string) (func() util.VersionedConfig, bool) {
	for _, version := range *v {
		if version.APIVersion == apiVersion {
			return version.Factory, true
		}
	}

	return nil, false
}

// IsSkaffoldConfig is for determining if a file is skaffold config file.
func IsSkaffoldConfig(file string) bool {
	if !kubernetes.HasKubernetesFileExtension(file) {
		return false
	}

	if config, err := ParseConfig(file); err == nil && config != nil {
		return true
	}
	return false
}

// ParseConfig reads a configuration file.
func ParseConfig(filename string) ([]util.VersionedConfig, error) {
	buf, err := misc.ReadConfiguration(filename)
	if err != nil {
		return nil, fmt.Errorf("read skaffold config: %w", err)
	}
	factories, err := configFactoryFromAPIVersion(buf)
	if err != nil {
		return nil, err
	}
	buf, err = removeYamlAnchors(buf)
	if err != nil {
		return nil, fmt.Errorf("unable to re-marshal YAML without dotted keys: %w", err)
	}
	return parseConfig(buf, factories)
}

// ParseConfigAndUpgrade reads a configuration file and upgrades it to a given version.
func ParseConfigAndUpgrade(filename, toVersion string) ([]util.VersionedConfig, error) {
	configs, err := ParseConfig(filename)
	if err != nil {
		return nil, err
	}

	// Check that the target version exists
	if _, present := SchemaVersions.Find(toVersion); !present {
		return nil, fmt.Errorf("unknown api version: %q", toVersion)
	}
	upgradeNeeded := false
	for _, cfg := range configs {
		// Check that the config's version is not newer than the target version
		currentVersion, err := apiversion.Parse(cfg.GetVersion())
		if err != nil {
			return nil, err
		}
		targetVersion, err := apiversion.Parse(toVersion)
		if err != nil {
			return nil, err
		}

		if currentVersion.NE(targetVersion) {
			upgradeNeeded = true
		}
		if currentVersion.GT(targetVersion) {
			return nil, fmt.Errorf("config version %q is more recent than target version %q: upgrade Skaffold", cfg.GetVersion(), toVersion)
		}
	}
	if !upgradeNeeded {
		return configs, nil
	}
	logrus.Debugf("config version out of date: upgrading to latest %q", toVersion)

	var upgraded []util.VersionedConfig
	for _, cfg := range configs {
		for cfg.GetVersion() != toVersion {
			cfg, err = cfg.Upgrade()
			if err != nil {
				return nil, fmt.Errorf("transforming skaffold config: %w", err)
			}
		}
		upgraded = append(upgraded, cfg)
	}
	return upgraded, nil
}

// configFactoryFromAPIVersion checks that all configs in the input stream have the same API version, and returns a function to create a config with that API version.
func configFactoryFromAPIVersion(buf []byte) ([]func() util.VersionedConfig, error) {
	// This is to quickly check that it's possibly a skaffold.yaml,
	// without parsing the whole file.
	if !bytes.Contains(buf, []byte("apiVersion")) {
		return nil, errors.New("missing apiVersion")
	}

	var factories []func() util.VersionedConfig
	b := bytes.NewReader(buf)
	decoder := yaml.NewDecoder(b)
	for {
		var v APIVersion
		err := decoder.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing api version: %w", err)
		}
		factory, present := SchemaVersions.Find(v.Version)
		if !present {
			return nil, fmt.Errorf("unknown api version: %q", v.Version)
		}
		factories = append(factories, factory)
	}
	return factories, nil
}

// removeYamlAnchors removes all top-level keys starting with `.` from the input stream so they can be used as YAML anchors
func removeYamlAnchors(buf []byte) ([]byte, error) {
	in := bytes.NewReader(buf)
	var out bytes.Buffer

	decoder := yaml.NewDecoder(in)
	decoder.KnownFields(true)
	encoder := yaml.NewEncoder(&out)
	for {
		parsed := make(map[string]interface{})
		err := decoder.Decode(parsed)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse YAML: %w", err)
		}
		for field := range parsed {
			if strings.HasPrefix(field, ".") {
				delete(parsed, field)
			}
		}
		err = encoder.Encode(parsed)
		if err != nil {
			return nil, err
		}
	}
	err := encoder.Close()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func parseConfig(buf []byte, factories []func() util.VersionedConfig) ([]util.VersionedConfig, error) {
	b := bytes.NewReader(buf)
	decoder := yaml.NewDecoder(b)
	decoder.KnownFields(true)
	var cfgs []util.VersionedConfig
	for index := 0; index < len(factories); index++ {
		cfg := factories[index]()
		err := decoder.Decode(cfg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse config: %w", err)
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}
