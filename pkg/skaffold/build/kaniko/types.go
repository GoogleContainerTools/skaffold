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

const (
	// BuildArgsFlag additional flag
	BuildArgsFlag = "--build-arg"
	// CacheFlag additional flag
	CacheFlag = "--cache"
	// CacheCopyLayersFlag additional flag
	CacheCopyLayersFlag = "--cache-copy-layers"
	// CacheRunLayersFlag additional flag
	CacheRunLayersFlag = "--cache-run-layers"
	// CacheDirFlag additional flag
	CacheDirFlag = "--cache-dir"
	// CacheRepoFlag additional flag
	CacheRepoFlag = "--cache-repo"
	// CacheTTLFlag additional flag
	CacheTTLFlag = "--cache-ttl"
	// TargetFlag additional flag
	TargetFlag = "--target"
	// CleanupFlag additional flag
	CleanupFlag = "--cleanup"
	// DigestFileFlag additional flag
	DigestFileFlag = "--digest-file"
	// Destination additional flag
	DestinationFlag = "--destination"
	// ForceFlag additional flag
	ForceFlag = "--force"
	// ImageFSExtractRetry additional flag
	ImageFSExtractRetryFlag = "--image-fs-extract-retry"
	// ImageNameWithDigestFileFlag  additional flag
	ImageNameWithDigestFileFlag = "--image-name-with-digest-file"
	// InsecureFlag additional flag
	InsecureFlag = "--insecure"
	// InsecurePullFlag additional flag
	InsecurePullFlag = "--insecure-pull"
	// InsecureRegistryFlag additional flag
	InsecureRegistryFlag = "--insecure-registry"
	// LabelFlag additional flag
	LabelFlag = "--label"
	// LogFormatFlag additional flag
	LogFormatFlag = "--log-format"
	// LogTimestampFlag additional flag
	LogTimestampFlag = "--log-timestamp"
	// OCILayoutFlag additional flag
	OCILayoutFlag = "--oci-layout-path"
	// RegistryCertificateFlag additional flag
	RegistryCertificateFlag = "--registry-certificate"
	// RegistryMirrorFlag additional flag
	RegistryMirrorFlag = "--registry-mirror"
	// ReproducibleFlag additional flag
	ReproducibleFlag = "--reproducible"
	// SingleSnapshotFlag additional flag
	SingleSnapshotFlag = "--single-snapshot"
	// SkipTLSFlag additional flag
	SkipTLSFlag = "--skip-tls-verify"
	// SkipTLSVerifyPullFlag additional flag
	SkipTLSVerifyPullFlag = "--skip-tls-verify-pull"
	// SkipTLSVerifyRegistryFlag additional flag
	SkipTLSVerifyRegistryFlag = "--skip-tls-verify-registry"
	// SkipUnusedStagesFlag additional flag
	SkipUnusedStagesFlag = "--skip-unused-stages"
	// SnapshotModeFlag additional flag
	SnapshotModeFlag = "--snapshot-mode"
	// PushRetryFlag additional flag
	PushRetryFlag = "--push-retry"
	// TarPathFlag additional flag
	TarPathFlag = "--tarPath"
	// UseNewRunFlag additional flag
	UseNewRunFlag = "--use-new-run"
	// VerbosityFlag additional flag
	VerbosityFlag = "--verbosity"
	// WhitelistVarRunFlag additional flag
	WhitelistVarRunFlag = "--whitelist-var-run"
	// DefaultImage is image used by the Kaniko pod by default
	DefaultImage = "gcr.io/kaniko-project/executor:latest"
	// DefaultSecretName for kaniko pod
	DefaultSecretName = "kaniko-secret"
	// DefaultTimeout for kaniko pod
	DefaultTimeout = "20m"
	// DefaultContainerName for kaniko pod
	DefaultContainerName = "kaniko"
	// DefaultEmptyDirName for kaniko pod
	DefaultEmptyDirName = "kaniko-emptydir"
	// DefaultEmptyDirMountPath for kaniko pod
	DefaultEmptyDirMountPath = "/kaniko/buildcontext"
	// DefaultCacheDirName for kaniko pod
	DefaultCacheDirName = "kaniko-cache"
	// DefaultCacheDirMountPath for kaniko pod
	DefaultCacheDirMountPath = "/cache"
	// DefaultDockerConfigSecretName for kaniko pod
	DefaultDockerConfigSecretName = "docker-cfg"
	// DefaultDockerConfigPath for kaniko pod
	DefaultDockerConfigPath = "/kaniko/.docker"
	// DefaultSecretMountPath for kaniko pod
	DefaultSecretMountPath = "/secret"
	// IgnorePathFlag additional flag
	IgnorePathFlag = "--ignore-path"
	// DefaultCopyMaxRetries for kaniko pod
	DefaultCopyMaxRetries = 3
	// DefaultCopyTimeout for kaniko pod
	DefaultCopyTimeout = "5m"
	// DefaultBuildContextCompressionLevel for kaniko pod
	DefaultBuildContextCompressionLevel = 1 // BestSpeed
)
