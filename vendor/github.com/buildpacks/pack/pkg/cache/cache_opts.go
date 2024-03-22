package cache

import (
	"encoding/csv"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Format int
type CacheInfo struct {
	Format Format
	Source string
}

type CacheOpts struct {
	Build  CacheInfo
	Launch CacheInfo
	Kaniko CacheInfo
}

const (
	CacheVolume Format = iota
	CacheImage
	CacheBind
)

func (f Format) String() string {
	switch f {
	case CacheImage:
		return "image"
	case CacheVolume:
		return "volume"
	case CacheBind:
		return "bind"
	}
	return ""
}

func (c *CacheInfo) SourceName() string {
	switch c.Format {
	case CacheImage:
		fallthrough
	case CacheVolume:
		return "name"
	case CacheBind:
		return "source"
	}
	return ""
}

func (c *CacheOpts) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	csvReader.Comma = ';'
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	cache := &c.Build
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return errors.Errorf("invalid field '%s' must be a key=value pair", field)
		}
		key := strings.ToLower(parts[0])
		value := strings.ToLower(parts[1])
		if key == "type" {
			switch value {
			case "build":
				cache = &c.Build
			case "launch":
				cache = &c.Launch
			default:
				return errors.Errorf("invalid cache type '%s'", value)
			}
			break
		}
	}

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return errors.Errorf("invalid field '%s' must be a key=value pair", field)
		}
		key := strings.ToLower(parts[0])
		value := strings.ToLower(parts[1])
		switch key {
		case "format":
			switch value {
			case "image":
				cache.Format = CacheImage
			case "volume":
				cache.Format = CacheVolume
			case "bind":
				cache.Format = CacheBind
			default:
				return errors.Errorf("invalid cache format '%s'", value)
			}
		case "name":
			cache.Source = value
		case "source":
			cache.Source = value
		}
	}

	err = sanitize(c)
	if err != nil {
		return err
	}
	return nil
}

func (c *CacheOpts) String() string {
	var cacheFlag string
	cacheFlag = fmt.Sprintf("type=build;format=%s;", c.Build.Format.String())
	if c.Build.Source != "" {
		cacheFlag += fmt.Sprintf("%s=%s;", c.Build.SourceName(), c.Build.Source)
	}

	cacheFlag += fmt.Sprintf("type=launch;format=%s;", c.Launch.Format.String())
	if c.Launch.Source != "" {
		cacheFlag += fmt.Sprintf("%s=%s;", c.Launch.SourceName(), c.Launch.Source)
	}

	return cacheFlag
}

func (c *CacheOpts) Type() string {
	return "cache"
}

func sanitize(c *CacheOpts) error {
	for _, v := range []CacheInfo{c.Build, c.Launch} {
		// volume cache name can be auto-generated
		if v.Format != CacheVolume && v.Source == "" {
			return errors.Errorf("cache '%s' is required", v.SourceName())
		}
	}

	var (
		resolvedPath string
		err          error
	)
	if c.Build.Format == CacheBind {
		if resolvedPath, err = filepath.Abs(c.Build.Source); err != nil {
			return errors.Wrap(err, "resolve absolute path")
		}
		c.Build.Source = filepath.Join(resolvedPath, "build-cache")
	}
	if c.Launch.Format == CacheBind {
		if resolvedPath, err = filepath.Abs(c.Launch.Source); err != nil {
			return errors.Wrap(err, "resolve absolute path")
		}
		c.Launch.Source = filepath.Join(resolvedPath, "launch-cache")
	}
	return nil
}
