package format

import "reflect"

type ObjToMapFunc func(i interface{}) map[string]interface{}

func DocToMap(inter interface{}, f ObjToMapFunc) (interface{}, int) {
	ik := reflect.TypeOf(inter).Kind()
	if ik == reflect.Ptr {
		return f(inter), 1
	}
	if ik != reflect.Slice {
		return nil, 0
	}
	v := reflect.ValueOf(inter)
	l := v.Len()
	count := 0
	ret := []map[string]interface{}{}
	for i := 0; i < l; i++ {
		if data := f(v.Index(i).Interface()); data != nil {
			ret = append(ret, data)
			count++
		}
	}
	return ret, count
}
