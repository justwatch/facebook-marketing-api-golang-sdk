package fb

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// appendJSON expects a pointer to a slice; in should contain a parsable JSON string,
// e.g. [1,2,3]. appendJSON will parse the content of in into v, if there is
// already some data in v, the data will be APPENDED.
func appendJSON(data []byte, v interface{}) (int, error) {
	tPtr := reflect.TypeOf(v)
	vPtr := reflect.ValueOf(v)

	if tPtr.Kind() != reflect.Ptr {
		return 0, fmt.Errorf("do not have ptr, got %s", tPtr)
	}

	if vPtr.Elem().Type().Kind() != reflect.Slice {
		return 0, fmt.Errorf("do not have ptr to slice, got ptr to %s", vPtr.Elem().Type().Kind())
	}

	slice := reflect.MakeSlice(reflect.SliceOf(tPtr.Elem().Elem()), 0, 0)
	slicePtr := reflect.New(slice.Type())
	slicePtr.Elem().Set(slice)
	dest := slicePtr.Interface()

	err := json.Unmarshal(data, dest)
	if err != nil {
		return 0, err
	}

	target := reflect.ValueOf(v).Elem()
	target = reflect.AppendSlice(target, reflect.ValueOf(dest).Elem())

	vPtr.Elem().Set(target)

	return slicePtr.Elem().Len(), nil
}
