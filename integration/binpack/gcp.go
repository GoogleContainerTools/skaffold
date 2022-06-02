/*
Copyright 2022 The Skaffold Authors

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

package binpack

// TODO(marlongamez): These timings are arbitrary right now for testing. Update when we have better information.

// GCPTimings contains the timings for tests marked with NeedsGcp
var GCPTimings = []Timing{
	{"TestBuildKanikoInsecureRegistry", 10.00},
	{"TestBuildKanikoWithExplicitRepo", 10.00},
	{"TestBuildInCluster", 10.00},
	{"TestBuildGCBWithExplicitRepo", 10.00},
	{"TestDeployCloudRun", 10.00},
	{"TestBuildDeploy", 10.00},
	{"TestDeployWithoutWorkspaces", 10.00},
	{"TestDevPortForwardGKELoadBalancer", 10.00},
	{"TestHelmDeploy", 10.00},
	{"TestRunGCPOnly", 10.00},
}

const MaxGCPBinTime = 31.0
