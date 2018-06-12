package types

import (
	"fmt"
	"time"
)

// UnsupportedProperties not yet supported by this implementation of the compose file
var UnsupportedProperties = []string{
	"build",
	"cap_add",
	"cap_drop",
	"cgroup_parent",
	"devices",
	"domainname",
	"external_links",
	"ipc",
	"links",
	"mac_address",
	"network_mode",
	"pid",
	"privileged",
	"restart",
	"security_opt",
	"shm_size",
	"sysctls",
	"ulimits",
	"userns_mode",
}

// DeprecatedProperties that were removed from the v3 format, but their
// use should not impact the behaviour of the application.
var DeprecatedProperties = map[string]string{
	"container_name": "Setting the container name is not supported.",
	"expose":         "Exposing ports is unnecessary - services on the same network can access each other's containers on any port.",
}

// ForbiddenProperties that are not supported in this implementation of the
// compose file.
var ForbiddenProperties = map[string]string{
	"extends":       "Support for `extends` is not implemented yet.",
	"volume_driver": "Instead of setting the volume driver on the service, define a volume using the top-level `volumes` option and specify the driver there.",
	"volumes_from":  "To share a volume between services, define it using the top-level `volumes` option and reference it from each service that shares it using the service-level `volumes` option.",
	"cpu_quota":     "Set resource limits using deploy.resources",
	"cpu_shares":    "Set resource limits using deploy.resources",
	"cpuset":        "Set resource limits using deploy.resources",
	"mem_limit":     "Set resource limits using deploy.resources",
	"memswap_limit": "Set resource limits using deploy.resources",
}

// ConfigFile is a filename and the contents of the file as a Dict
type ConfigFile struct {
	Filename string
	Config   map[string]interface{}
}

// ConfigDetails are the details about a group of ConfigFiles
type ConfigDetails struct {
	Version     string
	WorkingDir  string
	ConfigFiles []ConfigFile
	Environment map[string]string
}

// LookupEnv provides a lookup function for environment variables
func (cd ConfigDetails) LookupEnv(key string) (string, bool) {
	v, ok := cd.Environment[key]
	return v, ok
}

// Config is a full compose file configuration
type Config struct {
	Filename string `yaml:"-"`
	Version  string
	Services Services
	Networks map[string]NetworkConfig
	Volumes  map[string]VolumeConfig
	Secrets  map[string]SecretConfig
	Configs  map[string]ConfigObjConfig
}

// Services is a list of ServiceConfig
type Services []ServiceConfig

// MarshalYAML makes Services implement yaml.Marshaller
func (s Services) MarshalYAML() (interface{}, error) {
	services := map[string]ServiceConfig{}
	for _, service := range s {
		services[service.Name] = service
	}
	return services, nil
}

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name string `yaml:"-"`

	Build           BuildConfig                      `yaml:",omitempty"`
	CapAdd          []string                         `mapstructure:"cap_add" yaml:"cap_add,omitempty"`
	CapDrop         []string                         `mapstructure:"cap_drop" yaml:"cap_drop,omitempty"`
	CgroupParent    string                           `mapstructure:"cgroup_parent" yaml:"cgroup_parent,omitempty"`
	Command         ShellCommand                     `yaml:",omitempty"`
	Configs         []ServiceConfigObjConfig         `yaml:",omitempty"`
	ContainerName   string                           `mapstructure:"container_name" yaml:"container_name,omitempty"`
	CredentialSpec  CredentialSpecConfig             `mapstructure:"credential_spec" yaml:"credential_spec,omitempty"`
	DependsOn       []string                         `mapstructure:"depends_on" yaml:"depends_on,omitempty"`
	Deploy          DeployConfig                     `yaml:",omitempty"`
	Devices         []string                         `yaml:",omitempty"`
	DNS             StringList                       `yaml:",omitempty"`
	DNSSearch       StringList                       `mapstructure:"dns_search" yaml:"dns_search,omitempty"`
	DomainName      string                           `mapstructure:"domainname" yaml:"domainname,omitempty"`
	Entrypoint      ShellCommand                     `yaml:",omitempty"`
	Environment     MappingWithEquals                `yaml:",omitempty"`
	EnvFile         StringList                       `mapstructure:"env_file" yaml:"env_file,omitempty"`
	Expose          StringOrNumberList               `yaml:",omitempty"`
	ExternalLinks   []string                         `mapstructure:"external_links" yaml:"external_links,omitempty"`
	ExtraHosts      HostsList                        `mapstructure:"extra_hosts" yaml:"extra_hosts,omitempty"`
	Hostname        string                           `yaml:",omitempty"`
	HealthCheck     *HealthCheckConfig               `yaml:",omitempty"`
	Image           string                           `yaml:",omitempty"`
	Ipc             string                           `yaml:",omitempty"`
	Labels          Labels                           `yaml:",omitempty"`
	Links           []string                         `yaml:",omitempty"`
	Logging         *LoggingConfig                   `yaml:",omitempty"`
	MacAddress      string                           `mapstructure:"mac_address" yaml:"mac_address,omitempty"`
	NetworkMode     string                           `mapstructure:"network_mode" yaml:"network_mode,omitempty"`
	Networks        map[string]*ServiceNetworkConfig `yaml:",omitempty"`
	Pid             string                           `yaml:",omitempty"`
	Ports           []ServicePortConfig              `yaml:",omitempty"`
	Privileged      bool                             `yaml:",omitempty"`
	ReadOnly        bool                             `mapstructure:"read_only" yaml:"read_only,omitempty"`
	Restart         string                           `yaml:",omitempty"`
	Secrets         []ServiceSecretConfig            `yaml:",omitempty"`
	SecurityOpt     []string                         `mapstructure:"security_opt" yaml:"security_opt,omitempty"`
	StdinOpen       bool                             `mapstructure:"stdin_open" yaml:"stdin_open,omitempty"`
	StopGracePeriod *time.Duration                   `mapstructure:"stop_grace_period" yaml:"stop_grace_period,omitempty"`
	StopSignal      string                           `mapstructure:"stop_signal" yaml:"stop_signal,omitempty"`
	Tmpfs           StringList                       `yaml:",omitempty"`
	Tty             bool                             `mapstructure:"tty" yaml:"tty,omitempty"`
	Ulimits         map[string]*UlimitsConfig        `yaml:",omitempty"`
	User            string                           `yaml:",omitempty"`
	Volumes         []ServiceVolumeConfig            `yaml:",omitempty"`
	WorkingDir      string                           `mapstructure:"working_dir" yaml:"working_dir,omitempty"`
	Isolation       string                           `mapstructure:"isolation" yaml:"isolation,omitempty"`
}

// BuildConfig is a type for build
// using the same format at libcompose: https://github.com/docker/libcompose/blob/master/yaml/build.go#L12
type BuildConfig struct {
	Context    string            `yaml:",omitempty"`
	Dockerfile string            `yaml:",omitempty"`
	Args       MappingWithEquals `yaml:",omitempty"`
	Labels     Labels            `yaml:",omitempty"`
	CacheFrom  StringList        `mapstructure:"cache_from" yaml:"cache_from,omitempty"`
	Network    string            `yaml:",omitempty"`
	Target     string            `yaml:",omitempty"`
}

// ShellCommand is a string or list of string args
type ShellCommand []string

// StringList is a type for fields that can be a string or list of strings
type StringList []string

// StringOrNumberList is a type for fields that can be a list of strings or
// numbers
type StringOrNumberList []string

// MappingWithEquals is a mapping type that can be converted from a list of
// key[=value] strings.
// For the key with an empty value (`key=`), the mapped value is set to a pointer to `""`.
// For the key without value (`key`), the mapped value is set to nil.
type MappingWithEquals map[string]*string

// Labels is a mapping type for labels
type Labels map[string]string

// MappingWithColon is a mapping type that can be converted from a list of
// 'key: value' strings
type MappingWithColon map[string]string

// HostsList is a list of colon-separated host-ip mappings
type HostsList []string

// LoggingConfig the logging configuration for a service
type LoggingConfig struct {
	Driver  string            `yaml:",omitempty"`
	Options map[string]string `yaml:",omitempty"`
}

// DeployConfig the deployment configuration for a service
type DeployConfig struct {
	Mode           string         `yaml:",omitempty"`
	Replicas       *uint64        `yaml:",omitempty"`
	Labels         Labels         `yaml:",omitempty"`
	UpdateConfig   *UpdateConfig  `mapstructure:"update_config" yaml:"update_config,omitempty"`
	RollbackConfig *UpdateConfig  `mapstructure:"rollback_config" yaml:"rollback_config,omitempty"`
	Resources      Resources      `yaml:",omitempty"`
	RestartPolicy  *RestartPolicy `mapstructure:"restart_policy" yaml:"restart_policy,omitempty"`
	Placement      Placement      `yaml:",omitempty"`
	EndpointMode   string         `mapstructure:"endpoint_mode" yaml:"endpoint_mode,omitempty"`
}

// HealthCheckConfig the healthcheck configuration for a service
type HealthCheckConfig struct {
	Test        HealthCheckTest `yaml:",omitempty"`
	Timeout     *time.Duration  `yaml:",omitempty"`
	Interval    *time.Duration  `yaml:",omitempty"`
	Retries     *uint64         `yaml:",omitempty"`
	StartPeriod *time.Duration  `mapstructure:"start_period" yaml:"start_period,omitempty"`
	Disable     bool            `yaml:",omitempty"`
}

// HealthCheckTest is the command run to test the health of a service
type HealthCheckTest []string

// UpdateConfig the service update configuration
type UpdateConfig struct {
	Parallelism     *uint64       `yaml:",omitempty"`
	Delay           time.Duration `yaml:",omitempty"`
	FailureAction   string        `mapstructure:"failure_action" yaml:"failure_action,omitempty"`
	Monitor         time.Duration `yaml:",omitempty"`
	MaxFailureRatio float32       `mapstructure:"max_failure_ratio" yaml:"max_failure_ratio,omitempty"`
	Order           string        `yaml:",omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:",omitempty"`
	Reservations *Resource `yaml:",omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	// TODO: types to convert from units and ratios
	NanoCPUs         string            `mapstructure:"cpus" yaml:"cpus,omitempty"`
	MemoryBytes      UnitBytes         `mapstructure:"memory" yaml:"memory,omitempty"`
	GenericResources []GenericResource `mapstructure:"generic_resources" yaml:"generic_resources,omitempty"`
}

// GenericResource represents a "user defined" resource which can
// only be an integer (e.g: SSD=3) for a service
type GenericResource struct {
	DiscreteResourceSpec *DiscreteGenericResource `mapstructure:"discrete_resource_spec" yaml:"discrete_resource_spec,omitempty"`
}

// DiscreteGenericResource represents a "user defined" resource which is defined
// as an integer
// "Kind" is used to describe the Kind of a resource (e.g: "GPU", "FPGA", "SSD", ...)
// Value is used to count the resource (SSD=5, HDD=3, ...)
type DiscreteGenericResource struct {
	Kind  string
	Value int64
}

// UnitBytes is the bytes type
type UnitBytes int64

// MarshalYAML makes UnitBytes implement yaml.Marshaller
func (u UnitBytes) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%d", u), nil
}

// RestartPolicy the service restart policy
type RestartPolicy struct {
	Condition   string         `yaml:",omitempty"`
	Delay       *time.Duration `yaml:",omitempty"`
	MaxAttempts *uint64        `mapstructure:"max_attempts" yaml:"max_attempts,omitempty"`
	Window      *time.Duration `yaml:",omitempty"`
}

// Placement constraints for the service
type Placement struct {
	Constraints []string               `yaml:",omitempty"`
	Preferences []PlacementPreferences `yaml:",omitempty"`
}

// PlacementPreferences is the preferences for a service placement
type PlacementPreferences struct {
	Spread string `yaml:",omitempty"`
}

// ServiceNetworkConfig is the network configuration for a service
type ServiceNetworkConfig struct {
	Aliases     []string `yaml:",omitempty"`
	Ipv4Address string   `mapstructure:"ipv4_address" yaml:"ipv4_address,omitempty"`
	Ipv6Address string   `mapstructure:"ipv6_address" yaml:"ipv6_address,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      string `yaml:",omitempty"`
	Target    uint32 `yaml:",omitempty"`
	Published uint32 `yaml:",omitempty"`
	Protocol  string `yaml:",omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type        string               `yaml:",omitempty"`
	Source      string               `yaml:",omitempty"`
	Target      string               `yaml:",omitempty"`
	ReadOnly    bool                 `mapstructure:"read_only" yaml:"read_only,omitempty"`
	Consistency string               `yaml:",omitempty"`
	Bind        *ServiceVolumeBind   `yaml:",omitempty"`
	Volume      *ServiceVolumeVolume `yaml:",omitempty"`
	Tmpfs       *ServiceVolumeTmpfs  `yaml:",omitempty"`
}

// ServiceVolumeBind are options for a service volume of type bind
type ServiceVolumeBind struct {
	Propagation string `yaml:",omitempty"`
}

// ServiceVolumeVolume are options for a service volume of type volume
type ServiceVolumeVolume struct {
	NoCopy bool `mapstructure:"nocopy" yaml:"nocopy,omitempty"`
}

// ServiceVolumeTmpfs are options for a service volume of type tmpfs
type ServiceVolumeTmpfs struct {
	Size int64 `yaml:",omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string  `yaml:",omitempty"`
	Target string  `yaml:",omitempty"`
	UID    string  `yaml:",omitempty"`
	GID    string  `yaml:",omitempty"`
	Mode   *uint32 `yaml:",omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// UlimitsConfig the ulimit configuration
type UlimitsConfig struct {
	Single int `yaml:",omitempty"`
	Soft   int `yaml:",omitempty"`
	Hard   int `yaml:",omitempty"`
}

// MarshalYAML makes UlimitsConfig implement yaml.Marshaller
func (u *UlimitsConfig) MarshalYAML() (interface{}, error) {
	if u.Single != 0 {
		return u.Single, nil
	}
	return u, nil
}

// NetworkConfig for a network
type NetworkConfig struct {
	Name       string            `yaml:",omitempty"`
	Driver     string            `yaml:",omitempty"`
	DriverOpts map[string]string `mapstructure:"driver_opts" yaml:"driver_opts,omitempty"`
	Ipam       IPAMConfig        `yaml:",omitempty"`
	External   External          `yaml:",omitempty"`
	Internal   bool              `yaml:",omitempty"`
	Attachable bool              `yaml:",omitempty"`
	Labels     Labels            `yaml:",omitempty"`
}

// IPAMConfig for a network
type IPAMConfig struct {
	Driver string      `yaml:",omitempty"`
	Config []*IPAMPool `yaml:",omitempty"`
}

// IPAMPool for a network
type IPAMPool struct {
	Subnet string `yaml:",omitempty"`
}

// VolumeConfig for a volume
type VolumeConfig struct {
	Name       string            `yaml:",omitempty"`
	Driver     string            `yaml:",omitempty"`
	DriverOpts map[string]string `mapstructure:"driver_opts" yaml:"driver_opts,omitempty"`
	External   External          `yaml:",omitempty"`
	Labels     Labels            `yaml:",omitempty"`
}

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
// External.name is deprecated and replaced by Volume.name
type External struct {
	Name     string `yaml:",omitempty"`
	External bool   `yaml:",omitempty"`
}

// MarshalYAML makes External implement yaml.Marshaller
func (e External) MarshalYAML() (interface{}, error) {
	if e.Name == "" {
		return e.External, nil
	}
	return External{Name: e.Name}, nil
}

// CredentialSpecConfig for credential spec on Windows
type CredentialSpecConfig struct {
	File     string `yaml:",omitempty"`
	Registry string `yaml:",omitempty"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name     string   `yaml:",omitempty"`
	File     string   `yaml:",omitempty"`
	External External `yaml:",omitempty"`
	Labels   Labels   `yaml:",omitempty"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig
