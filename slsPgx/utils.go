package slsPgx

import (
	"errors"
	"reflect"
	"strings"
)

type statActivity struct {
	pid int
}

type connCred struct {
	database string
	host     string
	user     string
	url      string
}

func Int(value int) *int {
	return &value
}

func Bool(value bool) *bool {
	return &value
}

func Float32(value float32) *float32 {
	return &value
}

const (
	tooManyClientsErr        = "sorry, too many clients already"
	terminatingConnectionErr = "terminating connection due to administrator command"
)

var (
	connectionErrors = []string{tooManyClientsErr}
	queryErrors = []string{terminatingConnectionErr}
)

func containsError(s []string, e error) bool {
	for _, a := range s {
		if strings.Contains(e.Error(), a) {
			return true
		}
	}
	return false
}

func callFuncByName(myClass interface{}, funcName string, params ...interface{}) (reflect.Value, error) {
	myClassValue := reflect.ValueOf(myClass)
	m := myClassValue.MethodByName(funcName)
	if !m.IsValid() {
		return reflect.Value{}, errors.New("method not found: " + funcName)
	}
	in := make([]reflect.Value, len(params))
	for i, param := range params {
		in[i] = reflect.ValueOf(param)
	}
	res := m.Call(in)
	outputErr, ok := res[1].Interface().(error)
	if ok {
		return reflect.Value{}, outputErr
	}

	return res[0], nil
}
