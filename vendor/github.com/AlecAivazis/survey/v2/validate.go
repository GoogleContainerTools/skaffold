package survey

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/AlecAivazis/survey/v2/core"
)

// Required does not allow an empty value
func Required(val interface{}) error {
	// the reflect value of the result
	value := reflect.ValueOf(val)

	// if the value passed in is the zero value of the appropriate type
	if isZero(value) && value.Kind() != reflect.Bool {
		//lint:ignore ST1005 this error message should render as capitalized
		return errors.New("Value is required")
	}
	return nil
}

// MaxLength requires that the string is no longer than the specified value
func MaxLength(length int) Validator {
	// return a validator that checks the length of the string
	return func(val interface{}) error {
		if str, ok := val.(string); ok {
			// if the string is longer than the given value
			if len([]rune(str)) > length {
				// yell loudly
				return fmt.Errorf("value is too long. Max length is %v", length)
			}
		} else {
			// otherwise we cannot convert the value into a string and cannot enforce length
			return fmt.Errorf("cannot enforce length on response of type %v", reflect.TypeOf(val).Name())
		}

		// the input is fine
		return nil
	}
}

// MinLength requires that the string is longer or equal in length to the specified value
func MinLength(length int) Validator {
	// return a validator that checks the length of the string
	return func(val interface{}) error {
		if str, ok := val.(string); ok {
			// if the string is shorter than the given value
			if len([]rune(str)) < length {
				// yell loudly
				return fmt.Errorf("value is too short. Min length is %v", length)
			}
		} else {
			// otherwise we cannot convert the value into a string and cannot enforce length
			return fmt.Errorf("cannot enforce length on response of type %v", reflect.TypeOf(val).Name())
		}

		// the input is fine
		return nil
	}
}

// MaxItems requires that the list is no longer than the specified value
func MaxItems(numberItems int) Validator {
	// return a validator that checks the length of the list
	return func(val interface{}) error {
		if list, ok := val.([]core.OptionAnswer); ok {
			// if the list is longer than the given value
			if len(list) > numberItems {
				// yell loudly
				return fmt.Errorf("value is too long. Max items is %v", numberItems)
			}
		} else {
			// otherwise we cannot convert the value into a list of answer and cannot enforce length
			return fmt.Errorf("cannot impose the length on something other than a list of answers")
		}
		// the input is fine
		return nil
	}
}

// MinItems requires that the list is longer or equal in length to the specified value
func MinItems(numberItems int) Validator {
	// return a validator that checks the length of the list
	return func(val interface{}) error {
		if list, ok := val.([]core.OptionAnswer); ok {
			// if the list is shorter than the given value
			if len(list) < numberItems {
				// yell loudly
				return fmt.Errorf("value is too short. Min items is %v", numberItems)
			}
		} else {
			// otherwise we cannot convert the value into a list of answer and cannot enforce length
			return fmt.Errorf("cannot impose the length on something other than a list of answers")
		}
		// the input is fine
		return nil
	}
}

// ComposeValidators is a variadic function used to create one validator from many.
func ComposeValidators(validators ...Validator) Validator {
	// return a validator that calls each one sequentially
	return func(val interface{}) error {
		// execute each validator
		for _, validator := range validators {
			// if the answer's value is not valid
			if err := validator(val); err != nil {
				// return the error
				return err
			}
		}
		// we passed all validators, the answer is valid
		return nil
	}
}

// isZero returns true if the passed value is the zero object
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	}

	// compare the types directly with more general coverage
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}
