package cmd

import (
	"fmt"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type platformValue struct {
	platform *v1.Platform
}

func (pv *platformValue) Set(platform string) error {
	p, err := parsePlatform(platform)
	if err != nil {
		return err
	}
	pv.platform = p
	return nil
}

func (pv *platformValue) String() string {
	return platformToString(pv.platform)
}

func (pv *platformValue) Type() string {
	return "platform"
}

func platformToString(p *v1.Platform) string {
	if p == nil {
		return "all"
	}
	platform := ""
	if p.OS != "" && p.Architecture != "" {
		platform = p.OS + "/" + p.Architecture
	}
	if p.Variant != "" {
		platform += "/" + p.Variant
	}
	return platform
}

func parsePlatform(platform string) (*v1.Platform, error) {
	if platform == "all" {
		return nil, nil
	}

	p := &v1.Platform{}

	parts := strings.SplitN(platform, ":", 2)
	if len(parts) == 2 {
		p.OSVersion = parts[1]
	}

	parts = strings.Split(parts[0], "/")

	if len(parts) < 2 {
		return nil, fmt.Errorf("failed to parse platform '%s': expected format os/arch[/variant]", platform)
	}
	if len(parts) > 3 {
		return nil, fmt.Errorf("failed to parse platform '%s': too many slashes", platform)
	}

	p.OS = parts[0]
	p.Architecture = parts[1]
	if len(parts) > 2 {
		p.Variant = parts[2]
	}

	return p, nil
}
