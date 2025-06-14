package json

import gojson "encoding/json"

// UnmarshalGenericJSON unmarshal JSON into given generic type T
func UnmarshalGenericJSON[T any](b []byte) (v T, err error) {
	return v, gojson.Unmarshal(b, &v)
}
