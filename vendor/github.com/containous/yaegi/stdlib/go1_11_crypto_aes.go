// Code generated by 'goexports crypto/aes'. DO NOT EDIT.

// +build go1.11,!go1.12

package stdlib

import (
	"crypto/aes"
	"reflect"
)

func init() {
	Symbols["crypto/aes"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"BlockSize": reflect.ValueOf(aes.BlockSize),
		"NewCipher": reflect.ValueOf(aes.NewCipher),

		// type definitions
		"KeySizeError": reflect.ValueOf((*aes.KeySizeError)(nil)),
	}
}
