package float

import "strconv"

// ToFloat converts a value to float64, returns value, true if possible, 0, false otherwise
func ToFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// BoolToFloat converts a float to bool
func BoolToFloat(b bool) float64 {
	f := 0.0
	if b {
		f = 1.0
	}
	return f
}
