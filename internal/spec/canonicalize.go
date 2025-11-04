package spec

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zeebo/blake3"
)

// Canonicalize returns a canonical JSON representation of a feature
// with stable ordering for consistent hashing
func Canonicalize(feature Feature) ([]byte, error) {
	// Convert to map for stable key ordering
	data := map[string]interface{}{
		"id":       feature.ID,
		"title":    feature.Title,
		"desc":     feature.Desc,
		"priority": feature.Priority,
		"success":  feature.Success,
		"trace":    feature.Trace,
	}

	// Add API if present
	if len(feature.API) > 0 {
		apis := make([]map[string]interface{}, len(feature.API))
		for i, api := range feature.API {
			apiMap := map[string]interface{}{
				"method": api.Method,
				"path":   api.Path,
			}
			if api.Request != "" {
				apiMap["request"] = api.Request
			}
			if api.Response != "" {
				apiMap["response"] = api.Response
			}
			apis[i] = apiMap
		}
		data["api"] = apis
	}

	// Marshal with sorted keys
	return json.Marshal(sortKeys(data))
}

// Hash computes the blake3 hash of a canonicalized feature
func Hash(feature Feature) (string, error) {
	canonical, err := Canonicalize(feature)
	if err != nil {
		return "", fmt.Errorf("canonicalize feature: %w", err)
	}

	hasher := blake3.New()
	if _, err := hasher.Write(canonical); err != nil {
		return "", fmt.Errorf("hash feature: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// sortKeys recursively sorts map keys for stable JSON output
func sortKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sorted := make(map[string]interface{}, len(val))
		for _, k := range keys {
			sorted[k] = sortKeys(val[k])
		}
		return sorted

	case []interface{}:
		for i, item := range val {
			val[i] = sortKeys(item)
		}
		return val

	case []map[string]interface{}:
		for i, item := range val {
			val[i] = sortKeys(item).(map[string]interface{})
		}
		return val

	default:
		return v
	}
}
