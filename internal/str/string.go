package str

import (
	"strings"
)

// ValuesString concatenate all values of a map split by comma
func ValuesString(m map[string]string) string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return strings.Join(values, ", ")
}

// FindInSlice finds value in a slice
func FindInSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// AllContained determines if all a slice values are in the b slice
func AllContained(a, b []string) bool {
	lookup := make(map[string]struct{}, len(b))
	for _, v := range b {
		lookup[v] = struct{}{}
	}
	for _, item := range a {
		if _, ok := lookup[item]; !ok {
			return false
		}
	}
	return true
}

// ToSlice converts an input slice to string slice
func ToSlice(input []interface{}) ([]string, bool) {
	output := make([]string, len(input))
	for i, v := range input {
		s, ok := v.(string)
		if !ok {
			return nil, false
		}
		output[i] = s
	}
	return output, true
}
