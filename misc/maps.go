package misc

import (
	"errors"
	"reflect"
)

func StringKeysOrPanic(v interface{}) []string {
	retval, err := StringKeys(v)
	if err != nil {
		panic("parameter must be a map")
	}
	return retval
}

func StringKeys(v interface{}) ([]string, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, errors.New("Parameter must be a map")
	}
	t := rv.Type()
	if t.Key().Kind() != reflect.String {
		return nil, errors.New("Map key must be a string")
	}
	keys := rv.MapKeys()
	result := make([]string, len(keys))
	for i, kv := range rv.MapKeys() {
		result[i] = kv.String()
	}
	return result, nil
}
