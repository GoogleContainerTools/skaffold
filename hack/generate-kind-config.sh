#!/bin/sh
# Create a kind configuration to use the docker daemon's configured registry-mirrors.
docker system info --format '{{printf "apiVersion: kind.x-k8s.io/v1alpha4\nkind: Cluster\ncontainerdConfigPatches:\n"}}{{range $reg, $config := .RegistryConfig.IndexConfigs}}{{if $config.Mirrors}}{{printf "- |-\n  [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"%s\"]\n    endpoint = %q\n" $reg $config.Mirrors}}{{end}}{{end}}'
