package comparehelpers

import (
	"fmt"
	"reflect"
)

// Returns true if 'containee' is contained in 'container'
// Note this method searches all objects in 'container' for containee
// Contains is defined by the following relationship
// basic data types (string, float, int,...):
//
//	container == containee
//
// maps:
//
//	every key-value pair from containee is in container
//	Ex: {"a": 1, "b": 2, "c": 3} contains {"a": 1, "c": 3}
//
// arrays:
//
//	every element in containee is present and ordered in an array in container
//	Ex: [1, 1, 4, 3, 10, 4] contains [1, 3, 4 ]
//
// Limitaions:
// Cannot handle the following types: Pointers, Func
// Assumes we are compairing structs generated from JSON, YAML, or TOML.
func DeepContains(container, containee interface{}) bool {
	if container == nil || containee == nil {
		return container == containee
	}
	v1 := reflect.ValueOf(container)
	v2 := reflect.ValueOf(containee)

	return deepContains(v1, v2, 0)
}

func deepContains(v1, v2 reflect.Value, depth int) bool {
	if depth > 200 {
		panic("deep Contains depth exceeded, likely a circular reference")
	}
	if !v1.IsValid() || !v2.IsValid() {
		return v1.IsValid() == v2.IsValid()
	}

	switch v1.Kind() {
	case reflect.Array, reflect.Slice:
		// check for subset matches in arrays
		return arrayLikeContains(v1, v2, depth+1)
	case reflect.Map:
		return mapContains(v1, v2, depth+1)
	case reflect.Interface:
		return deepContains(v1.Elem(), v2, depth+1)
	case reflect.Ptr, reflect.Struct, reflect.Func:
		panic(fmt.Sprintf("unimplmemented comparison for type: %s", v1.Kind().String()))
	default: // assume it is a atomic datatype
		return reflect.DeepEqual(v1, v2)
	}
}

func mapContains(v1, v2 reflect.Value, depth int) bool {
	t2 := v2.Kind()
	if t2 == reflect.Interface {
		return mapContains(v1, v2.Elem(), depth+1)
	} else if t2 == reflect.Map {
		result := true
		for _, k := range v2.MapKeys() {
			k2Val := v2.MapIndex(k)
			k1Val := v1.MapIndex(k)
			if !k1Val.IsValid() || !reflect.DeepEqual(k1Val.Interface(), k2Val.Interface()) {
				result = false
				break
			}
		}
		if result {
			return true
		}
	}
	for _, k := range v1.MapKeys() {
		val := v1.MapIndex(k)
		if deepContains(val, v2, depth+1) {
			return true
		}
	}
	return false
}

func arrayLikeContains(v1, v2 reflect.Value, depth int) bool {
	t2 := v2.Kind()
	if t2 == reflect.Interface {
		return mapContains(v1, v2.Elem(), depth+1)
	} else if t2 == reflect.Array || t2 == reflect.Slice {
		v1Index := 0
		v2Index := 0
		for v1Index < v1.Len() && v2Index < v2.Len() {
			if reflect.DeepEqual(v1.Index(v1Index).Interface(), v2.Index(v2Index).Interface()) {
				v2Index++
			}
			v1Index++
		}
		if v2Index == v2.Len() {
			return true
		}
	}
	for i := 0; i < v1.Len(); i++ {
		if deepContains(v1.Index(i), v2, depth+1) {
			return true
		}
	}
	return false
}
