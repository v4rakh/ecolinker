package tm

import (
	"errors"
	"time"
)

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// TimeRanges calculates time ranges between start to end with given skipInterval
// if endOffset is non-zero, add the duration to each calculated end range (can be negative!)
// if addTrailingRange is true, adds a range from last calculated end time to desired even if difference between these do not fully cover skipInterval
func TimeRanges(start, end time.Time, skipInterval, endOffset time.Duration, addTrailingRange bool) ([]TimeRange, error) {
	if start.IsZero() || end.IsZero() || skipInterval == 0 {
		return nil, errors.New("assert: nil values are not allowed")
	}
	if !start.Before(end) {
		return nil, errors.New("assert: start must be before end")
	}

	var ranges []TimeRange
	current := start

	for current.Before(end) {
		next := current.Add(skipInterval)
		if next.After(end) {
			next = end
		}

		rangeEnd := next.Add(endOffset)
		ranges = append(ranges, TimeRange{Start: current, End: rangeEnd})
		current = next
	}

	if addTrailingRange && len(ranges) > 0 && ranges[len(ranges)-1].End.Before(end) {
		ranges = append(ranges, TimeRange{Start: ranges[len(ranges)-1].End, End: end})
	}

	return ranges, nil
}
