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

package environments

import (
	"github.com/hashicorp/go-hclog"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugins/environments/gcb"
)

//TODO(this should be rather wired through an env var from the skaffold process)
const DefaultPluginLogLevel = hclog.Info

// SkaffoldCoreEnvPluginExecutionMap maps the core plugin name to the execution function
var SkaffoldCoreEnvPluginExecutionMap = map[string]func() error{
	"googlecloudbuild": gcb.Execute(DefaultPluginLogLevel),
}

// Execute executes a plugin - does not validate
func Execute(plugin string) error {
	return SkaffoldCoreEnvPluginExecutionMap[plugin]()
}
