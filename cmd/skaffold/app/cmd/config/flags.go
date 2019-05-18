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

package config

import "github.com/spf13/pflag"

var configFile, kubecontext string
var showAll, global bool

func AddConfigFlags(f *pflag.FlagSet) {
	f.StringVarP(&configFile, "config", "c", "", "Path to Skaffold config")
	f.StringVarP(&kubecontext, "kube-context", "k", "", "Kubectl context to set values against")
}

func AddSetFlags(f *pflag.FlagSet) {
	f.BoolVarP(&global, "global", "g", false, "Set value for global config")
}
