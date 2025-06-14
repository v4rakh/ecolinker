//go:build !integration

package str

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractValuesFromString(t *testing.T) {
	a := assert.New(t)
	a.Equal("", ValuesString(nil))
	a.Equal("val1", ValuesString(map[string]string{"key1": "val1"}))
	valuesString := ValuesString(map[string]string{"key1": "val1", "key2": "val2"})
	a.Contains(valuesString, "val1")
	a.Contains(valuesString, "val2")
}

func TestContains(t *testing.T) {
	a := assert.New(t)
	a.True(Contains([]string{""}, ""))
	a.True(Contains([]string{"test", "abc"}, "test"))
	a.False(Contains([]string{"abc"}, "test"))
	a.False(Contains([]string{""}, "test"))
}

func TestAllContained(t *testing.T) {
	a := assert.New(t)
	a.True(AllContained([]string{"a", "b"}, []string{"a", "b", "c"}))
	a.True(AllContained([]string{"b", "a"}, []string{"a", "b", "c"}))
	a.True(AllContained([]string{"b", "c", "a"}, []string{"a", "b", "c"}))
	a.False(AllContained([]string{"c", "a"}, []string{"a"}))
	a.True(AllContained([]string{}, []string{"a"}))
	a.True(AllContained([]string{}, []string{}))
	a.False(AllContained([]string{"a"}, []string{}))
}

func TestEqualsIgnoreOrder(t *testing.T) {
	a := assert.New(t)
	a.True(EqualsIgnoreOrder([]string{"c", "a", "b", "a"}, []string{"a", "a", "b", "c"}))
	a.False(EqualsIgnoreOrder([]string{"c", "a", "z", "a"}, []string{"a", "a", "b", "c"}))
	a.False(EqualsIgnoreOrder([]string{"c", "a", "z", "a"}, []string{"a", "a", "b", "c"}))
	a.True(EqualsIgnoreOrder([]string{}, []string{}))
	a.False(EqualsIgnoreOrder([]string{"a"}, []string{}))
	a.False(EqualsIgnoreOrder([]string{}, []string{"a"}))
}

func TestFindInSlice(t *testing.T) {
	a := assert.New(t)
	a.True(FindInSlice([]string{""}, ""))
	a.True(FindInSlice([]string{"test", "abc"}, "test"))
	a.False(FindInSlice([]string{"abc"}, "test"))
	a.False(FindInSlice([]string{""}, "test"))
}
