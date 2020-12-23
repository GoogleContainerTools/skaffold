/*
Copyright 2020 The Skaffold Authors

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

package kaniko

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Args returns kaniko command arguments
func Args(artifact *latest.KanikoArtifact, tag, context string) ([]string, error) {
	args := []string{
		"--destination", tag,
		"--dockerfile", artifact.DockerfilePath,
	}

	if context != "" {
		args = append(args, "--context", context)
	}

	buildArgs, err := util.MapToFlag(artifact.BuildArgs, BuildArgsFlag)
	if err != nil {
		return args, err
	}

	args = append(args, buildArgs...)

	if artifact.Cache != nil {
		args = append(args, CacheFlag)

		if artifact.Cache.Repo != "" {
			args = append(args, CacheRepoFlag, artifact.Cache.Repo)
		}
		if artifact.Cache.HostPath != "" {
			args = append(args, CacheDirFlag, artifact.Cache.HostPath)
		}
		if artifact.Cache.TTL != "" {
			args = append(args, CacheTTLFlag, artifact.Cache.TTL)
		}
	}

	if artifact.Target != "" {
		args = append(args, TargetFlag, artifact.Target)
	}

	if artifact.Cleanup {
		args = append(args, CleanupFlag)
	}

	if artifact.DigestFile != "" {
		args = append(args, DigestFileFlag, artifact.DigestFile)
	}

	if artifact.Force {
		args = append(args, ForceFlag)
	}

	if artifact.ImageNameWithDigestFile != "" {
		args = append(args, ImageNameWithDigestFileFlag, artifact.ImageNameWithDigestFile)
	}

	if artifact.Insecure {
		args = append(args, InsecureFlag)
	}

	if artifact.InsecurePull {
		args = append(args, InsecurePullFlag)
	}

	if artifact.LogFormat != "" {
		args = append(args, LogFormatFlag, artifact.LogFormat)
	}

	if artifact.LogTimestamp {
		args = append(args, LogTimestampFlag)
	}

	if artifact.NoPush {
		args = append(args, NoPushFlag)
	}

	if artifact.OCILayoutPath != "" {
		args = append(args, OCILayoutFlag, artifact.OCILayoutPath)
	}

	if artifact.RegistryMirror != "" {
		args = append(args, RegistryMirrorFlag, artifact.RegistryMirror)
	}

	if artifact.Reproducible {
		args = append(args, ReproducibleFlag)
	}

	if artifact.SingleSnapshot {
		args = append(args, SingleSnapshotFlag)
	}

	if artifact.SkipTLS {
		args = append(args, SkipTLSFlag)
		reg, err := artifactRegistry(tag)
		if err != nil {
			return nil, err
		}
		args = append(args, SkipTLSVerifyRegistryFlag, reg)
	}

	if artifact.SkipTLSVerifyPull {
		args = append(args, SkipTLSVerifyPullFlag)
	}

	if artifact.SkipUnusedStages {
		args = append(args, SkipUnusedStagesFlag)
	}

	if artifact.SnapshotMode != "" {
		args = append(args, SnapshotModeFlag, artifact.SnapshotMode)
	}

	if artifact.TarPath != "" {
		args = append(args, TarPathFlag, artifact.TarPath)
	}

	if artifact.UseNewRun {
		args = append(args, UseNewRunFlag)
	}

	if artifact.Verbosity != "" {
		args = append(args, VerbosityFlag, artifact.Verbosity)
	}

	if artifact.WhitelistVarRun {
		args = append(args, WhitelistVarRunFlag)
	}

	var iRegArgs []string
	for _, r := range artifact.InsecureRegistry {
		iRegArgs = append(iRegArgs, InsecureRegistryFlag, r)
	}
	args = append(args, iRegArgs...)

	var sRegArgs []string
	for _, r := range artifact.SkipTLSVerifyRegistry {
		sRegArgs = append(sRegArgs, SkipTLSVerifyRegistryFlag, r)
	}
	args = append(args, sRegArgs...)

	registryCertificate, err := util.MapToFlag(artifact.RegistryCertificate, RegistryCertificateFlag)
	if err != nil {
		return args, err
	}

	args = append(args, registryCertificate...)

	labels, err := util.MapToFlag(artifact.Label, LabelFlag)
	if err != nil {
		return args, err
	}

	args = append(args, labels...)

	return args, nil
}

func artifactRegistry(i string) (string, error) {
	ref, err := name.ParseReference(i)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve registry url from artifact: %w", err)
	}
	return ref.Context().RegistryStr(), nil
}
