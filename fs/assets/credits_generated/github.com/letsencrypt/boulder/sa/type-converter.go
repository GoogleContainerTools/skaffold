package sa

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp/v3"
	jose "gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/identifier"
)

// BoulderTypeConverter is used by Gorp for storing objects in DB.
type BoulderTypeConverter struct{}

// ToDb converts a Boulder object to one suitable for the DB representation.
func (tc BoulderTypeConverter) ToDb(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case identifier.ACMEIdentifier, []core.Challenge, []string, [][]int:
		jsonBytes, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		return string(jsonBytes), nil
	case jose.JSONWebKey:
		jsonBytes, err := t.MarshalJSON()
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	case core.AcmeStatus:
		return string(t), nil
	case core.OCSPStatus:
		return string(t), nil
	default:
		return val, nil
	}
}

// FromDb converts a DB representation back into a Boulder object.
func (tc BoulderTypeConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch target.(type) {
	case *identifier.ACMEIdentifier, *[]core.Challenge, *[]string, *[][]int:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("FromDb: Unable to convert *string")
			}
			b := []byte(*s)
			err := json.Unmarshal(b, target)
			if err != nil {
				return badJSONError(
					fmt.Sprintf("binder failed to unmarshal %T", target),
					b,
					err)
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(string), Target: target, Binder: binder}, true
	case *jose.JSONWebKey:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *string", holder)
			}
			if *s == "" {
				return errors.New("FromDb: Empty JWK field.")
			}
			b := []byte(*s)
			k, ok := target.(*jose.JSONWebKey)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *jose.JSONWebKey", target)
			}
			err := k.UnmarshalJSON(b)
			if err != nil {
				return badJSONError(
					"binder failed to unmarshal JWK",
					b,
					err)
			}
			return nil
		}
		return gorp.CustomScanner{Holder: new(string), Target: target, Binder: binder}, true
	case *core.AcmeStatus:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *string", holder)
			}
			st, ok := target.(*core.AcmeStatus)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *core.AcmeStatus", target)
			}

			*st = core.AcmeStatus(*s)
			return nil
		}
		return gorp.CustomScanner{Holder: new(string), Target: target, Binder: binder}, true
	case *core.OCSPStatus:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *string", holder)
			}
			st, ok := target.(*core.OCSPStatus)
			if !ok {
				return fmt.Errorf("FromDb: Unable to convert %T to *core.OCSPStatus", target)
			}

			*st = core.OCSPStatus(*s)
			return nil
		}
		return gorp.CustomScanner{Holder: new(string), Target: target, Binder: binder}, true
	default:
		return gorp.CustomScanner{}, false
	}
}
