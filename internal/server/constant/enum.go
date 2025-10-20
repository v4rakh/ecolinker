//go:generate go-enum --marshal --mustparse --values --names --output-suffix _generated

package constant

// ENUM(json, console)
type ConfigLogEncoding string

// ENUM(epoch, epochmillis, epochnanos, iso8601, rfc3339, rfc3339nano)
type ConfigLogTimeEncoder string

// ENUM(postgres)
type ConfigDatabaseType string

// ENUM(none, basic_single, basic_credentials)
type ConfigAuthMode string

// ENUM(other, powerocean)
type DeviceKind string

// ENUM(quota, status)
type TopicKind string

// ENUM(device_parameters, device_historical_data)
type CollectorKind string

// ENUM(daily, weekly)
type HistoricalDataStep string

// FromVariadicToStr converts variadic notation to string array if type is of string
func FromVariadicToStr[T ~string](s ...T) []string {
	arr := make([]string, 0, len(s))
	for _, i := range s {
		arr = append(arr, string(i))
	}
	return arr
}
