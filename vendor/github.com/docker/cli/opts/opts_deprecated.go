package opts

import "github.com/docker/cli/opts/swarmopts"

// PortOpt represents a port config in swarm mode.
//
// Deprecated: use [swarmopts.PortOpt]
type PortOpt = swarmopts.PortOpt

// ConfigOpt is a Value type for parsing configs.
//
// Deprecated: use [swarmopts.ConfigOpt]
type ConfigOpt = swarmopts.ConfigOpt

// SecretOpt is a Value type for parsing secrets
//
// Deprecated: use [swarmopts.SecretOpt]
type SecretOpt = swarmopts.SecretOpt
