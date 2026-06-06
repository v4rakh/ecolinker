package float

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertFloat(t *testing.T) {
	a := assert.New(t)

	var res float64
	var ok bool

	res, ok = ToFloat(nil)
	a.False(ok)
	a.Equal(0.0, res)

	res, ok = ToFloat([]string{"test"})
	a.False(ok)
	a.Equal(0.0, res)

	res, ok = ToFloat("0.5")
	a.True(ok)
	a.Equal(0.5, res)

	res, ok = ToFloat(-42)
	a.True(ok)
	a.Equal(-42.0, res)

	res, ok = ToFloat(false)
	a.True(ok)
	a.Equal(0.0, res)

	res, ok = ToFloat(true)
	a.True(ok)
	a.Equal(1.0, res)
}
