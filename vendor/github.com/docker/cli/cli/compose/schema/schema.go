// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	defaultVersion = "3.13"
	versionField   = "version"
)

type portsFormatChecker struct{}

func (portsFormatChecker) Validate(input any) error {
	switch input.(type) {
	case string, json.Number,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		_, err := nat.ParsePortSpec(fmt.Sprint(input))
		return err
	default:
		// Formats only apply to supported string/number values.
		return nil
	}
}

type durationFormatChecker struct{}

func (durationFormatChecker) Validate(input any) error {
	value, ok := input.(string)
	if !ok {
		return nil
	}
	_, err := time.ParseDuration(value)
	return err
}

// Version returns the version of the config, defaulting to the latest "3.x"
// version (3.13). If only the major version "3" is specified, it is used as
// version "3.x" and returns the default version (latest 3.x).
func Version(config map[string]any) string {
	version, ok := config[versionField]
	if !ok {
		return defaultVersion
	}
	return normalizeVersion(fmt.Sprintf("%v", version))
}

func normalizeVersion(version string) string {
	switch version {
	case "", "3":
		return defaultVersion
	default:
		return version
	}
}

//go:embed data/config_schema_v*.json
var schemas embed.FS

// Validate uses the jsonschema to validate the configuration
func Validate(config map[string]any, version string) error {
	version = normalizeVersion(version)
	schemaData, err := schemas.ReadFile("data/config_schema_v" + version + ".json")
	if err != nil {
		return fmt.Errorf("unsupported Compose file version: %s", version)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return err
	}

	compiler := jsonschema.NewCompiler()
	compiler.RegisterFormat(&jsonschema.Format{Name: "ports", Validate: portsFormatChecker{}.Validate})
	compiler.RegisterFormat(&jsonschema.Format{Name: "duration", Validate: durationFormatChecker{}.Validate})
	if err := compiler.AddResource("schema.json", schemaDoc); err != nil {
		return err
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return err
	}

	if err := schema.Validate(toJSONValue(config)); err != nil {
		if validationErr, ok := errors.AsType[*jsonschema.ValidationError](err); ok {
			return getMostSpecificError(validationErr)
		}
		return err
	}

	return nil
}

func toJSONValue(v any) any {
	if v == nil {
		return nil
	}

	switch value := reflect.ValueOf(v); value.Kind() { //nolint: exhaustive // only need to handle maps and slices.
	case reflect.Map:
		result := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result[iter.Key().String()] = toJSONValue(iter.Value().Interface())
		}
		return result
	case reflect.Slice:
		result := make([]any, value.Len())
		for i := range value.Len() {
			result[i] = toJSONValue(value.Index(i).Interface())
		}
		return result
	default:
		return v
	}
}

var printer = message.NewPrinter(language.English)

func getDescription(err validationError) string {
	switch parent := err.parent.ErrorKind.(type) {
	case *kind.Type:
		types := make([]string, len(parent.Want))
		for i, typ := range parent.Want {
			types[i] = humanReadableType(typ)
		}
		return "must be " + strings.Join(types, " or ")
	case *kind.AnyOf, *kind.OneOf:
		if err.child != nil {
			return getDescription(validationError{parent: err.child})
		}
		return err.parent.ErrorKind.LocalizedString(printer)
	case *kind.AdditionalProperties:
		if len(parent.Properties) == 1 {
			return fmt.Sprintf("additional property '%s' is not allowed", parent.Properties[0])
		}
		return err.parent.ErrorKind.LocalizedString(printer)
	case *kind.Minimum:
		want, _ := parent.Want.Float64()
		return fmt.Sprintf("must be greater than or equal to %v", want)
	default:
		return err.parent.ErrorKind.LocalizedString(printer)
	}
}

func humanReadableType(definition string) string {
	if definition[0:1] == "[" {
		allTypes := strings.Split(definition[1:len(definition)-1], ",")
		for i, t := range allTypes {
			allTypes[i] = humanReadableType(t)
		}
		return fmt.Sprintf(
			"%s or %s",
			strings.Join(allTypes[0:len(allTypes)-1], ", "),
			allTypes[len(allTypes)-1],
		)
	}
	switch definition {
	case "object":
		return "a mapping"
	case "array":
		return "a list"
	case "integer":
		return "an integer"
	case "null":
		return "null"
	default:
		return "a " + definition
	}
}

type validationError struct {
	parent *jsonschema.ValidationError
	child  *jsonschema.ValidationError
}

func (err validationError) Error() string {
	validationErr := err.parent
	if err.child != nil {
		validationErr = err.child
	}
	field := strings.Join(validationErr.InstanceLocation, ".")
	if field == "" {
		field = "(root)"
	}
	return field + ": " + getDescription(err)
}

func getMostSpecificError(result *jsonschema.ValidationError) validationError {
	errs := flattenErrors(result)
	mostSpecificError := 0
	for i, err := range errs {
		if specificity(err) > specificity(errs[mostSpecificError]) {
			mostSpecificError = i
			continue
		}

		if specificity(err) == specificity(errs[mostSpecificError]) {
			// Invalid type errors win in a tie-breaker for most specific field name.
			if isTypeError(err) && !isTypeError(errs[mostSpecificError]) {
				mostSpecificError = i
			}
		}
	}

	err := validationError{parent: errs[mostSpecificError]}
	switch err.parent.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		err.child = mostSpecificCause(err.parent)
	}
	return err
}

func mostSpecificCause(err *jsonschema.ValidationError) *jsonschema.ValidationError {
	var best *jsonschema.ValidationError

	for _, cause := range err.Causes {
		for _, candidate := range flattenErrors(cause) {
			switch candidate.ErrorKind.(type) {
			case *kind.OneOf, *kind.AnyOf:
				if nested := mostSpecificCause(candidate); nested != nil {
					candidate = nested
				}
			}

			if best == nil || specificity(candidate) > specificity(best) {
				best = candidate
				continue
			}

			if specificity(candidate) == specificity(best) {
				// Within oneOf/anyOf, an error from an alternative whose
				// type matched is more useful than a type mismatch from
				// another alternative.
				if !isTypeError(candidate) && isTypeError(best) {
					best = candidate
				}
			}
		}
	}

	return best
}

func flattenErrors(err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	var errs []*jsonschema.ValidationError
	var walk func(*jsonschema.ValidationError)
	walk = func(err *jsonschema.ValidationError) {
		switch err.ErrorKind.(type) {
		case *kind.Schema, *kind.Group, *kind.Reference:
			for _, cause := range err.Causes {
				walk(cause)
			}
		default:
			errs = append(errs, err)
		}
	}
	walk(err)
	return errs
}

func isTypeError(err *jsonschema.ValidationError) bool {
	_, ok := err.ErrorKind.(*kind.Type)
	return ok
}

func specificity(err *jsonschema.ValidationError) int {
	return len(err.InstanceLocation)
}
