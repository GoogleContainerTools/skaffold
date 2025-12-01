package swarmopts

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/sirupsen/logrus"
)

const (
	portOptTargetPort    = "target"
	portOptPublishedPort = "published"
	portOptProtocol      = "protocol"
	portOptMode          = "mode"
)

// PortOpt represents a port config in swarm mode.
type PortOpt struct {
	ports []swarm.PortConfig
}

// Set a new port value
//
//nolint:gocyclo
func (p *PortOpt) Set(value string) error {
	longSyntax, err := regexp.MatchString(`\w+=\w+(,\w+=\w+)*`, value)
	if err != nil {
		return err
	}
	if longSyntax {
		csvReader := csv.NewReader(strings.NewReader(value))
		fields, err := csvReader.Read()
		if err != nil {
			return err
		}

		pConfig := swarm.PortConfig{
			Protocol:    network.TCP,
			PublishMode: swarm.PortConfigPublishModeIngress,
		}
		for _, field := range fields {
			// TODO(thaJeztah): these options should not be case-insensitive.
			key, val, ok := strings.Cut(strings.ToLower(field), "=")
			if !ok || key == "" {
				return fmt.Errorf("invalid field: %s", field)
			}
			switch key {
			case portOptProtocol:
				switch proto := network.IPProtocol(val); proto {
				case network.TCP, network.UDP, network.SCTP:
					pConfig.Protocol = proto
				default:
					return fmt.Errorf("invalid protocol value '%s'", val)
				}
			case portOptMode:
				switch swarm.PortConfigPublishMode(val) {
				case swarm.PortConfigPublishModeIngress, swarm.PortConfigPublishModeHost:
					pConfig.PublishMode = swarm.PortConfigPublishMode(val)
				default:
					return fmt.Errorf("invalid publish mode value (%s): must be either '%s' or '%s'", val, swarm.PortConfigPublishModeIngress, swarm.PortConfigPublishModeHost)
				}
			case portOptTargetPort:
				tPort, err := strconv.ParseUint(val, 10, 16)
				if err != nil {
					var numErr *strconv.NumError
					if errors.As(err, &numErr) {
						err = numErr.Err
					}
					return fmt.Errorf("invalid target port (%s): value must be an integer: %w", val, err)
				}

				pConfig.TargetPort = uint32(tPort)
			case portOptPublishedPort:
				pPort, err := strconv.ParseUint(val, 10, 16)
				if err != nil {
					var numErr *strconv.NumError
					if errors.As(err, &numErr) {
						err = numErr.Err
					}
					return fmt.Errorf("invalid published port (%s): value must be an integer: %w", val, err)
				}

				pConfig.PublishedPort = uint32(pPort)
			default:
				return fmt.Errorf("invalid field key: %s", key)
			}
		}

		if pConfig.TargetPort == 0 {
			return fmt.Errorf("missing mandatory field '%s'", portOptTargetPort)
		}

		p.ports = append(p.ports, pConfig)
	} else {
		// short syntax
		ports, portBindingMap, err := nat.ParsePortSpecs([]string{value})
		if err != nil {
			return err
		}
		for _, portBindings := range portBindingMap {
			for _, portBinding := range portBindings {
				if portBinding.HostIP != "" {
					return errors.New("hostip is not supported")
				}
			}
		}

		var portConfigs []swarm.PortConfig
		for port := range ports {
			portProto, err := network.ParsePort(string(port))
			if err != nil {
				return err
			}
			portConfig, err := ConvertPortToPortConfig(portProto, portBindingMap)
			if err != nil {
				return err
			}
			portConfigs = append(portConfigs, portConfig...)
		}
		p.ports = append(p.ports, portConfigs...)
	}
	return nil
}

// Type returns the type of this option
func (*PortOpt) Type() string {
	return "port"
}

// String returns a string repr of this option
func (p *PortOpt) String() string {
	ports := make([]string, 0, len(p.ports))
	for _, port := range p.ports {
		repr := fmt.Sprintf("%v:%v/%s/%s", port.PublishedPort, port.TargetPort, port.Protocol, port.PublishMode)
		ports = append(ports, repr)
	}
	return strings.Join(ports, ", ")
}

// Value returns the ports
func (p *PortOpt) Value() []swarm.PortConfig {
	return p.ports
}

// ConvertPortToPortConfig converts ports to the swarm type
func ConvertPortToPortConfig(
	portProto network.Port,
	portBindings map[nat.Port][]nat.PortBinding,
) ([]swarm.PortConfig, error) {
	ports := make([]swarm.PortConfig, 0, len(portBindings))
	for _, binding := range portBindings[nat.Port(portProto.String())] {
		if p := net.ParseIP(binding.HostIP); p != nil && !p.IsUnspecified() {
			// TODO(thaJeztah): use context-logger, so that this output can be suppressed (in tests).
			logrus.Warnf("ignoring IP-address (%s:%s) service will listen on '0.0.0.0'", net.JoinHostPort(binding.HostIP, binding.HostPort), portProto.String())
		}

		startHostPort, endHostPort, err := nat.ParsePortRange(binding.HostPort)

		if err != nil && binding.HostPort != "" {
			return nil, fmt.Errorf("invalid hostport binding (%s) for port (%d)", binding.HostPort, portProto.Num())
		}

		for i := startHostPort; i <= endHostPort; i++ {
			ports = append(ports, swarm.PortConfig{
				// TODO Name: ?
				Protocol:      portProto.Proto(),
				TargetPort:    uint32(portProto.Num()),
				PublishedPort: uint32(i),
				PublishMode:   swarm.PortConfigPublishModeIngress,
			})
		}
	}
	return ports, nil
}
