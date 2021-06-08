#!/bin/sh

# Copyright 2019 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Create a kind configuration to use the docker daemon's configured registry-mirrors.
docker system info --format '{{printf "apiVersion: kind.x-k8s.io/v1alpha4\nkind: Cluster\ncontainerdConfigPatches:\n"}}{{range $reg, $config := .RegistryConfig.IndexConfigs}}{{if $config.Mirrors}}{{printf "- |-\n  [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"%s\"]\n    endpoint = [" $reg}}{{range $index, $mirror := $config.Mirrors}}{{if $index}},{{end}}{{printf "%q" $mirror}}{{end}}{{printf "]\n"}}{{end}}{{end}}'
