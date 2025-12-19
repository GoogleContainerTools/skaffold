package opts

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// GpuOpts is a Value type for parsing mounts
type GpuOpts struct {
	values []container.DeviceRequest
}

func parseCount(s string) (int, error) {
	if s == "all" {
		return -1, nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		var numErr *strconv.NumError
		if errors.As(err, &numErr) {
			err = numErr.Err
		}
		return 0, fmt.Errorf(`invalid count (%s): value must be either "all" or an integer: %w`, s, err)
	}
	return i, nil
}

// Set a new mount value
//
//nolint:gocyclo
func (o *GpuOpts) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	req := container.DeviceRequest{}

	seen := map[string]struct{}{}
	// Set writable as the default
	for _, field := range fields {
		key, val, withValue := strings.Cut(field, "=")
		if _, ok := seen[key]; ok {
			return fmt.Errorf("gpu request key '%s' can be specified only once", key)
		}
		seen[key] = struct{}{}

		if !withValue {
			seen["count"] = struct{}{}
			req.Count, err = parseCount(key)
			if err != nil {
				return err
			}
			continue
		}

		switch key {
		case "driver":
			req.Driver = val
		case "count":
			req.Count, err = parseCount(val)
			if err != nil {
				return err
			}
		case "device":
			req.DeviceIDs = strings.Split(val, ",")
		case "capabilities":
			req.Capabilities = [][]string{append(strings.Split(val, ","), "gpu")}
		case "options":
			r := csv.NewReader(strings.NewReader(val))
			optFields, err := r.Read()
			if err != nil {
				return fmt.Errorf("failed to read gpu options: %w", err)
			}
			req.Options = ConvertKVStringsToMap(optFields)
		default:
			return fmt.Errorf("unexpected key '%s' in '%s'", key, field)
		}
	}

	if _, ok := seen["count"]; !ok && req.DeviceIDs == nil {
		req.Count = 1
	}
	if req.Options == nil {
		req.Options = make(map[string]string)
	}
	if req.Capabilities == nil {
		req.Capabilities = [][]string{{"gpu"}}
	}

	o.values = append(o.values, req)
	return nil
}

// Type returns the type of this option
func (*GpuOpts) Type() string {
	return "gpu-request"
}

// String returns a string repr of this option
func (o *GpuOpts) String() string {
	gpus := []string{}
	for _, gpu := range o.values {
		gpus = append(gpus, fmt.Sprintf("%v", gpu))
	}
	return strings.Join(gpus, ", ")
}

// Value returns the mounts
func (o *GpuOpts) Value() []container.DeviceRequest {
	return o.values
}
