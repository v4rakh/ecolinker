package str

import (
	"sort"
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

// EqualsIgnoreOrder compares slices ignoring order
func EqualsIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

// Contains determines if value is in a slice
func Contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
