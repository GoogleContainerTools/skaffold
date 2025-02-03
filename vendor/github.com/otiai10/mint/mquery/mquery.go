package mquery

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Query(m interface{}, q string) interface{} {
	return query(m, strings.Split(q, "."))
}

func query(m interface{}, qs []string) interface{} {
	t := reflect.TypeOf(m)
	switch t.Kind() {
	case reflect.Map:
		return queryMap(m, t, qs)
	case reflect.Slice:
		return querySlice(m, t, qs)
	default:
		return m
	}
}

func queryMap(m interface{}, t reflect.Type, qs []string) interface{} {
	if len(qs) == 0 {
		return m
	}
	val := reflect.ValueOf(m)
	if val.IsZero() {
		return nil
	}
	switch t.Key().Kind() {
	case reflect.String:
		val := reflect.ValueOf(m).MapIndex(reflect.ValueOf(qs[0]))
		if !val.IsValid() {
			return nil
		}
		return query(val.Interface(), qs[1:])
	case reflect.Int:
		i, err := strconv.Atoi(qs[0])
		if err != nil {
			return fmt.Errorf("cannot access map with keyword: %s: %v", qs[0], err)
		}
		val := reflect.ValueOf(m).MapIndex(reflect.ValueOf(i))
		if !val.IsValid() {
			return nil
		}
		return query(val.Interface(), qs[1:])
	}
	return nil
}

func querySlice(m interface{}, t reflect.Type, qs []string) interface{} {
	if len(qs) == 0 {
		return m
	}
	v := reflect.ValueOf(m)
	if v.Len() == 0 {
		return nil
	}
	i, err := strconv.Atoi(qs[0])
	if err != nil {
		return fmt.Errorf("cannot access slice with keyword: %s: %v", qs[0], err)
	}
	if v.Len() <= i {
		return nil
	}
	next := v.Index(i).Interface()
	return query(next, qs[1:])
}
