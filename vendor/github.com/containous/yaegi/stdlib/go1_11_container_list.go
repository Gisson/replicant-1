// Code generated by 'goexports container/list'. DO NOT EDIT.

// +build go1.11,!go1.12

package stdlib

import (
	"container/list"
	"reflect"
)

func init() {
	Symbols["container/list"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"New": reflect.ValueOf(list.New),

		// type definitions
		"Element": reflect.ValueOf((*list.Element)(nil)),
		"List":    reflect.ValueOf((*list.List)(nil)),
	}
}