package util

import (
	jsoniter "github.com/json-iterator/go"
)

// JSON is a faster drop-in replacement for encoding/json
var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

// ToJSONB converts any interface to a byte slice for JSONB storage
func ToJSONB(v interface{}) []byte {
	b, _ := JSON.Marshal(v)
	return b
}
