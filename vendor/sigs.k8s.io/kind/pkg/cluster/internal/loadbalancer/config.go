/*
Copyright 2019 The Kubernetes Authors.

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

package loadbalancer

import (
	"bytes"
	"fmt"
	"net"
	"text/template"

	"sigs.k8s.io/kind/pkg/errors"
)

// ConfigData is supplied to the loadbalancer config template
type ConfigData struct {
	ControlPlanePort int
	BackendServers   map[string]string
	IPv6             bool
}

// DynamicFilesystemConfigTemplate holds the Envoy bootstrap configuration template
// for file-based dynamic xDS (CDS and LDS).
// https://www.envoyproxy.io/docs/envoy/latest/start/quick-start/configuration-dynamic-filesystem
const DynamicFilesystemConfigTemplate = `
node:
  cluster: %s
  id: %s

dynamic_resources:
  cds_config:
    resource_api_version: V3
    path_config_source:
      path: %s
  lds_config:
    resource_api_version: V3
    path_config_source:
      path: %s

admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 10000
`

// keep in sync with dynamicFilesystemConfig
const (
	ProxyConfigPathCDS = "/home/envoy/cds.yaml"
	ProxyConfigPathLDS = "/home/envoy/lds.yaml"
	ProxyConfigPath    = "/home/envoy/envoy.yaml"
	ProxyConfigDir     = "/home/envoy"
)

// ProxyLDSConfigTemplate is the loadbalancer config template for listeners
const ProxyLDSConfigTemplate = `
resources:
- "@type": type.googleapis.com/envoy.config.listener.v3.Listener
  name: listener_apiserver
  address:
    socket_address:
      address: {{ if .IPv6 }}"::"{{ else }}"0.0.0.0"{{ end }}
      port_value: {{ .ControlPlanePort }}
  filter_chains:
  - filters:
    - name: envoy.filters.network.tcp_proxy
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
        stat_prefix: ingress_tcp
        cluster: kube_apiservers
`

// ProxyCDSConfigTemplate is the loadbalancer config template for clusters
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/health_check.proto#envoy-v3-api-msg-config-core-v3-healthcheck-httphealthcheck
const ProxyCDSConfigTemplate = `
resources:
- "@type": type.googleapis.com/envoy.config.cluster.v3.Cluster
  name: kube_apiservers
  connect_timeout: 0.25s
  type: STRICT_DNS
  lb_policy: ROUND_ROBIN
  dns_lookup_family: {{ if $.IPv6 -}} AUTO {{- else -}} V4_PREFERRED {{- end }}
  health_checks:
  - timeout: 3s
    interval: 2s
    unhealthy_threshold: 2
    healthy_threshold: 1
    initial_jitter: 0s
    no_traffic_interval: 3s
    always_log_health_check_failures: true
    always_log_health_check_success: true
    event_log_path: /dev/stdout
    http_health_check:
      path: /healthz
    transport_socket_match_criteria:
      tls_mode: "true"
  transport_socket_matches:
  - name: "health_check_tls"
    match:
      tls_mode: "true"
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        common_tls_context:
          validation_context:
            trust_chain_verification: ACCEPT_UNTRUSTED
  load_assignment:
    cluster_name: kube_apiservers
    endpoints:
    - lb_endpoints:
{{- range $server, $address := .BackendServers }}
{{- $hp := hostPort $address }}
      - endpoint:
          address:
            socket_address:
              address: {{ $hp.host }}
              port_value: {{ $hp.port }}
{{- end }}
`

func hostPort(addr string) (map[string]string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	return map[string]string{"host": host, "port": port}, nil
}

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data *ConfigData, configTemplate string) (config string, err error) {
	funcs := template.FuncMap{
		"hostPort": hostPort,
	}
	t, err := template.New("loadbalancer-config").Funcs(funcs).Parse(configTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}
	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}

	return buff.String(), nil
}

func GenerateBootstrapCommand(clusterName, containerName string) []string {
	// populate the values to dynamic template
	envoyConfig := fmt.Sprintf(
		DynamicFilesystemConfigTemplate,
		clusterName,   // node.cluster = Kind cluster name
		containerName, // node.id = container name
		ProxyConfigPathCDS,
		ProxyConfigPathLDS,
	)

	// Create dynamic Envoy config files and start Envoy with retry,
	// since it has an initialization phase before forwarding traffic.
	// cmd := []string{"bash", "-c",
	// 	fmt.Sprintf(`mkdir -p %s && echo -en '%s' > %s && touch %s && touch %s && while true; do envoy -c %s && break; sleep 1; done`, constants.ProxyConfigDir,
	// 		envoyConfig, constants.ProxyConfigPath, constants.ProxyConfigPathCDS, constants.ProxyConfigPathLDS, constants.ProxyConfigPath)}
	// Create dynamic Envoy config files with valid empty resources
	emptyConfig := "resources: []"
	return []string{"bash", "-c",
		fmt.Sprintf(`mkdir -p %s && echo -en '%s' > %s && echo -en '%s' > %s && echo -en '%s' > %s && while true; do envoy -c %s && break; sleep 1; done`,
			ProxyConfigDir,
			envoyConfig, ProxyConfigPath,
			emptyConfig, ProxyConfigPathCDS, // Initialize CDS
			emptyConfig, ProxyConfigPathLDS, // Initialize LDS
			ProxyConfigPath)}
}
