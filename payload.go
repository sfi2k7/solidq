package solidq

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strconv"
)

type Interface interface{}
type Payload map[string]Interface

func (p Payload) Get(k string) Interface {
	v := p[k]
	return v
}

func (p Payload) Set(k string, v Interface) {
	p[k] = v
}

func (p Payload) Json() string {
	bts, err := json.Marshal(p)
	if err != nil {
		fmt.Println("Error in Payload JSON()", err)
	}
	return string(bts)
}

func PayloadFromString(v string) *Payload {
	var single Payload
	json.Unmarshal([]byte(v), &single)
	return &single
}

func PayloadFromMapString(m map[string]string) Payload {
	var data = Payload{}
	if m == nil {
		return data
	}

	for k, v := range m {
		data[k] = v
	}
	return data
}

func (p Payload) Remove(k string) {
	delete(p, k)
}

func (p Payload) HasKey(k string) bool {
	_, ok := p[k]
	return ok
}

func (p Payload) ParseData(k string, target interface{}) error {
	str := p.DataAsString(k)
	if len(str) == 0 {
		return errors.New("no data")
	}

	return json.Unmarshal([]byte(str), target)
}

func (p Payload) Bool(k string) bool {
	v := p.Get(k)

	if b, ok := v.(bool); ok {
		// fmt.Println("Returning bool", b)
		return b
	}

	if str, ok := v.(string); ok {
		if len(str) == 0 {
			return false
		}
		// fmt.Println("Parsing as bool", str)
		b, err := strconv.ParseBool(str)
		if err != nil {
			fmt.Println("Error parding bool")
		}
		return b
	}
	// fmt.Println("Returning false (default)")
	return false
}

func (p Payload) Clone() Payload {
	cloned := Payload{}
	for k, v := range p {
		cloned[k] = v
	}
	return cloned
}

func (p Payload) DataAsString(k string) string {
	data, ok := p[k]
	if !ok {
		return ""
	}

	fmt.Println(reflect.TypeOf(data).String())
	switch tp := data.(type) {
	case nil:
		return ""
	case map[string]interface{}:
		b, _ := json.Marshal(tp)
		return string(b)
	case []map[string]interface{}:
		b, _ := json.Marshal(tp)
		return string(b)
	case []interface{}:
		b, _ := json.Marshal(tp)
		return string(b)
	case string:
		return tp
	case interface{}:
		b, _ := json.Marshal(tp)
		return string(b)
	default:
		return fmt.Sprint(tp)
	}
}

func (p Payload) String(k string) string {
	v := p[k]
	if v == nil {
		return ""
	}

	str, ok := v.(string)
	if ok {
		return str
	}
	return fmt.Sprint(v)
}

func (p Payload) ArrayString(k string) []string {
	var result []string
	switch tp := p[k].(type) {
	case []string:
		result = tp
	case []interface{}:
		for _, i := range tp {
			s, ok := i.(string)
			if ok {
				result = append(result, s)
			}
		}
	case []int:
	case []int64:
	case []float64:
		for _, i := range tp {
			result = append(result, fmt.Sprint(i))
		}
	}

	return result
}

func (p Payload) Int(k string) int {
	// fmt.Println("WD", wd)
	v := p[k]
	// fmt.Println("p V", v)
	if v == nil {
		fmt.Println("Int:V us nil")
		return 0
	}

	switch vt := v.(type) {
	case int:
		return vt
	case int64:
		return int(vt)
	case float64:
		return int(vt)
	default:
		return -1
	}
}

func (p Payload) Join(src Payload) Payload {
	maps.Copy(p, src)
	return p
}
