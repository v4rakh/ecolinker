package str

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractValuesFromString(t *testing.T) {
	a := assert.New(t)
	a.Equal("", ValuesString(nil))
	a.Equal("val1", ValuesString(map[string]string{"key1": "val1"}))
	valuesString := ValuesString(map[string]string{"key1": "val1", "key2": "val2"})
	a.Contains(valuesString, "val1")
	a.Contains(valuesString, "val2")
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

func TestFindInSlice(t *testing.T) {
	a := assert.New(t)
	a.True(FindInSlice([]string{""}, ""))
	a.True(FindInSlice([]string{"test", "abc"}, "test"))
	a.False(FindInSlice([]string{"abc"}, "test"))
	a.False(FindInSlice([]string{""}, "test"))
}

func TestToSliceAllStrings(t *testing.T) {
	a := assert.New(t)

	input := []interface{}{"foo", "bar", "baz"}
	expected := []string{"foo", "bar", "baz"}
	result, ok := ToSlice(input)
	a.True(ok)
	a.Equal(expected, result)
}

func TestToSliceMixedTypes(t *testing.T) {
	a := assert.New(t)

	input := []interface{}{"foo", 123, "baz"}
	result, ok := ToSlice(input)
	a.False(ok)
	a.Nil(result)
}

func TestToSliceEmptySlice(t *testing.T) {
	a := assert.New(t)

	input := []interface{}{}
	expected := []string{}
	result, ok := ToSlice(input)
	a.True(ok)
	a.Equal(expected, result)
}

func TestToSliceNilInput(t *testing.T) {
	a := assert.New(t)

	var input []interface{}
	var expected []string
	result, ok := ToSlice(input)
	a.True(ok)
	a.NotEqual(expected, result)
}
