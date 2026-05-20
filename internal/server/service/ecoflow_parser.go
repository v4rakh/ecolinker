package service

import (
	"fmt"
	"regexp"
	"strconv"
)

type stringIndexValueMap struct {
	Key     string         `json:"key"`
	Indices map[string]int `json:"indices"`
	Value   interface{}    `json:"value"`
}

// extractIndicesAndValueList transforms indices and values from a map to list
func extractIndicesAndValueList(input map[string]interface{}) []stringIndexValueMap {
	re := regexp.MustCompile(`_(\d+)_?`)
	reMultiUnderscore := regexp.MustCompile(`_+`)
	reTrim := regexp.MustCompile(`^_+|_+$`)
	result := make([]stringIndexValueMap, 0, len(input))

	for rawKey, val := range input {
		indices := make(map[string]int)
		cleaned := rawKey

		matches := re.FindAllStringSubmatchIndex(rawKey, -1)
		offset := 0
		for i, match := range matches {
			digitStr := rawKey[match[2]:match[3]]
			digit, _ := strconv.Atoi(digitStr)
			indices[fmt.Sprintf("index%d", i)] = digit

			start := match[0] - offset
			end := match[1] - offset
			if cleaned[start:end][len(cleaned[start:end])-1] == '_' && end < len(cleaned) {
				cleaned = cleaned[:start] + cleaned[end-1:]
				offset += end - start - 1
			} else {
				cleaned = cleaned[:start] + cleaned[end:]
				offset += end - start
			}
		}

		cleaned = reMultiUnderscore.ReplaceAllString(cleaned, "_")
		cleaned = reTrim.ReplaceAllString(cleaned, "")

		result = append(result, stringIndexValueMap{
			Key:     cleaned,
			Indices: indices,
			Value:   val,
		})
	}
	return result
}

func flatten(in interface{}, out map[string]interface{}) {
	flattenPrefix("", in, out)
}

func flattenPrefix(prefix string, in interface{}, out map[string]interface{}) {
	switch v := in.(type) {
	case map[string]interface{}:
		for k, val := range v {
			key := k
			if prefix != "" {
				key = prefix + "_" + k
			}
			flattenPrefix(key, val, out)
		}
	case []interface{}:
		for i, val := range v {
			key := prefix + "_" + strconv.Itoa(i)
			flattenPrefix(key, val, out)
		}
	default:
		out[prefix] = v
	}
}
