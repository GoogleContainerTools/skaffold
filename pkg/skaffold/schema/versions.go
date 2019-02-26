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
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	apiversion "github.com/GoogleContainerTools/skaffold/pkg/skaffold/apiversion"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha5"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta5"
	misc "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

type APIVersion struct {
	Version string `yaml:"apiVersion"`
}

var SchemaVersions = Versions{
	{v1alpha1.Version, v1alpha1.NewSkaffoldPipeline},
	{v1alpha2.Version, v1alpha2.NewSkaffoldPipeline},
	{v1alpha3.Version, v1alpha3.NewSkaffoldPipeline},
	{v1alpha4.Version, v1alpha4.NewSkaffoldPipeline},
	{v1alpha5.Version, v1alpha5.NewSkaffoldPipeline},
	{v1beta1.Version, v1beta1.NewSkaffoldPipeline},
	{v1beta2.Version, v1beta2.NewSkaffoldPipeline},
	{v1beta3.Version, v1beta3.NewSkaffoldPipeline},
	{v1beta4.Version, v1beta4.NewSkaffoldPipeline},
	{v1beta5.Version, v1beta5.NewSkaffoldPipeline},
	{latest.Version, latest.NewSkaffoldPipeline},
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

// ParseConfig reads a configuration file.
func ParseConfig(filename string, upgrade bool) (util.VersionedConfig, error) {
	buf, err := misc.ReadConfiguration(filename)
	if err != nil {
		return nil, errors.Wrap(err, "read skaffold config")
	}

	apiVersion := &APIVersion{}
	if err := yaml.Unmarshal(buf, apiVersion); err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	factory, present := SchemaVersions.Find(apiVersion.Version)
	if !present {
		return nil, errors.Errorf("unknown api version: '%s'", apiVersion.Version)
	}

	cfg := factory()
	if err := yaml.UnmarshalStrict(buf, cfg); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}

	if err := yamltags.ProcessStruct(cfg); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	if upgrade && cfg.GetVersion() != latest.Version {
		cfg, err = upgradeToLatest(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// upgradeToLatest upgrades a configuration to the latest version.
func upgradeToLatest(vc util.VersionedConfig) (util.VersionedConfig, error) {
	var err error

	// first, check to make sure config version isn't too new
	version, err := apiversion.Parse(vc.GetVersion())
	if err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	semver := apiversion.MustParse(latest.Version)
	if version.EQ(semver) {
		return vc, nil
	}
	if version.GT(semver) {
		return nil, fmt.Errorf("config version %s is too new for this version: upgrade Skaffold", vc.GetVersion())
	}

	logrus.Warnf("config version (%s) out of date: upgrading to latest (%s)", vc.GetVersion(), latest.Version)

	for vc.GetVersion() != latest.Version {
		vc, err = vc.Upgrade()
		if err != nil {
			return nil, errors.Wrapf(err, "transforming skaffold config")
		}
	}

	return vc, nil
}
