/*
Copyright 2020 The Knative Authors

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

package configmap

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ParseFunc is a function taking ConfigMap data and applying a parse operation to it.
type ParseFunc func(map[string]string) error

// AsString passes the value at key through into the target, if it exists.
func AsString(key string, target *string) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			*target = raw
		}
		return nil
	}
}

// AsBool parses the value at key as a boolean into the target, if it exists.
func AsBool(key string, target *bool) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseBool(raw)
			*target = val // If err != nil — this is always false.
			return err
		}
		return nil
	}
}

// AsInt32 parses the value at key as an int32 into the target, if it exists.
func AsInt32(key string, target *int32) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = int32(val)
		}
		return nil
	}
}

// AsInt64 parses the value at key as an int64 into the target, if it exists.
func AsInt64(key string, target *int64) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = val
		}
		return nil
	}
}

// AsInt parses the value at key as an int into the target, if it exists.
func AsInt(key string, target *int) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = val
		}
		return nil
	}
}

// AsUint32 parses the value at key as an uint32 into the target, if it exists.
func AsUint32(key string, target *uint32) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = uint32(val)
		}
		return nil
	}
}

// AsFloat64 parses the value at key as a float64 into the target, if it exists.
func AsFloat64(key string, target *float64) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = val
		}
		return nil
	}
}

// AsDuration parses the value at key as a time.Duration into the target, if it exists.
func AsDuration(key string, target *time.Duration) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}
			*target = val
		}
		return nil
	}
}

// AsStringSet parses the value at key as a sets.String (split by ',') into the target, if it exists.
func AsStringSet(key string, target *sets.String) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			*target = sets.NewString(strings.Split(raw, ",")...)
		}
		return nil
	}
}

// AsQuantity parses the value at key as a *resource.Quantity into the target, if it exists
func AsQuantity(key string, target **resource.Quantity) ParseFunc {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := resource.ParseQuantity(raw)
			if err != nil {
				return fmt.Errorf("failed to parse %q: %w", key, err)
			}

			*target = &val
		}
		return nil
	}
}

// AsOptionalNamespacedName parses the value at key as a types.NamespacedName into the target, if it exists
// The namespace and name are both required and expected to be valid DNS labels
func AsOptionalNamespacedName(key string, target **types.NamespacedName) ParseFunc {
	return func(data map[string]string) error {
		if _, ok := data[key]; !ok {
			return nil
		}

		*target = &types.NamespacedName{}
		return AsNamespacedName(key, *target)(data)
	}
}

// AsNamespacedName parses the value at key as a types.NamespacedName into the target, if it exists
// The namespace and name are both required and expected to be valid DNS labels
func AsNamespacedName(key string, target *types.NamespacedName) ParseFunc {
	return func(data map[string]string) error {
		raw, ok := data[key]
		if !ok {
			return nil
		}

		v := strings.SplitN(raw, string(types.Separator), 3)

		if len(v) != 2 {
			return fmt.Errorf("failed to parse %q: expected 'namespace/name' format", key)
		}

		for _, val := range v {
			if errs := validation.ValidateNamespaceName(val, false); len(errs) > 0 {
				return fmt.Errorf("failed to parse %q: %s", key, strings.Join(errs, ", "))
			}
		}

		target.Namespace = v[0]
		target.Name = v[1]

		return nil
	}
}

// Parse parses the given map using the parser functions passed in.
func Parse(data map[string]string, parsers ...ParseFunc) error {
	for _, parse := range parsers {
		if err := parse(data); err != nil {
			return err
		}
	}
	return nil
}
