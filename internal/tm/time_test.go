package tm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimePairsAssertNil(t *testing.T) {
	a := assert.New(t)

	_, err := TimeRanges(time.Now(), time.Now(), 0, 0, false)
	a.ErrorContains(err, "assert: nil values are not allowed")

	zeroTime := new(time.Time)

	_, err = TimeRanges(time.Now(), *zeroTime, time.Hour, 0, false)
	a.ErrorContains(err, "assert: nil values are not allowed")

	_, err = TimeRanges(*zeroTime, time.Now(), time.Hour, 0, false)
	a.ErrorContains(err, "assert: nil values are not allowed")
}

func TestTimePairsAssertEqual(t *testing.T) {
	a := assert.New(t)

	start := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	_, err := TimeRanges(start, end, time.Hour, 0, false)
	a.ErrorContains(err, "assert: start must be before end")
}

func TestTimePairsAssertEarlier(t *testing.T) {
	a := assert.New(t)

	start := time.Date(2025, 10, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)

	_, err := TimeRanges(start, end, time.Hour, 0, false)
	a.ErrorContains(err, "assert: start must be before end")
}

func TestTimePairs(t *testing.T) {
	a := assert.New(t)

	start := time.Date(2025, 05, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 21, 0, 0, 0, 0, time.UTC)
	diff := time.Hour * 24
	ranges, err := TimeRanges(start, end, diff, -1*time.Second, false)

	expected := int(end.Sub(start).Abs().Hours() / 24)
	a.NoError(err)
	a.Equal(expected, len(ranges))
}
