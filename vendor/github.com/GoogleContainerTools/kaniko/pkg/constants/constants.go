/*
Copyright 2018 Google LLC

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

package constants

const (
	// DefaultLogLevel is the default log level
	DefaultLogLevel = "info"

	// RootDir is the path to the root directory
	RootDir = "/"

	// WorkspaceDir is the path to the workspace directory
	WorkspaceDir = "/workspace"

	//KanikoDir is the path to the Kaniko directory
	KanikoDir = "/kaniko"

	WhitelistPath = "/proc/self/mountinfo"

	Author = "kaniko"

	// ContextTar is the default name of the tar uploaded to GCS buckets
	ContextTar = "context.tar.gz"

	// BuildContextDir is the directory a build context will be unpacked into,
	// for example, a tarball from a GCS bucket will be unpacked here
	BuildContextDir = "/kaniko/buildcontext/"

	// KanikoIntermediateStagesDir is where we will store intermediate stages
	// as tarballs in case they are needed later on
	KanikoIntermediateStagesDir = "/kaniko/stages"

	// Various snapshot modes:
	SnapshotModeTime = "time"
	SnapshotModeFull = "full"

	// NoBaseImage is the scratch image
	NoBaseImage = "scratch"

	GCSBuildContextPrefix      = "gs://"
	S3BuildContextPrefix       = "s3://"
	LocalDirBuildContextPrefix = "dir://"

	HOME = "HOME"
	// DefaultHOMEValue is the default value Docker sets for $HOME
	DefaultHOMEValue = "/root"

	// Docker command names
	Cmd        = "cmd"
	Entrypoint = "entrypoint"
)

// KanikoBuildFiles is the list of files required to build kaniko
var KanikoBuildFiles = []string{"/kaniko/executor",
	"/kaniko/ssl/certs/ca-certificates.crt",
	"/kaniko/docker-credential-gcr",
	"/kaniko/docker-credential-ecr-login",
	"/kaniko/.docker/config.json"}

// ScratchEnvVars are the default environment variables needed for a scratch image.
var ScratchEnvVars = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
