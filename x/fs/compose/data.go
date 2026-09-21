package compose

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
)

func (file File) normalized() (File, error) {
	out := File{Type: file.Type, Mode: file.Mode, Values: maps.Clone(file.Values)}
	if file.Data == nil {
		return out, nil
	}
	normalized, err := normalize(file.Data)
	if err != nil {
		return File{}, err
	}
	data, ok := normalized.(map[string]any)
	if !ok {
		return File{}, ErrData
	}
	out.Data = data
	return out, nil
}

func normalize(value any) (any, error) {
	switch typed := value.(type) {
	case nil, string, bool:
		return typed, nil
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer, nil
		}
		float, err := typed.Float64()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrData, typed.String())
		}
		return normalizeFloat(float)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			normalized, err := normalize(child)
			if err != nil {
				return nil, err
			}
			out[index] = normalized
		}
		return out, nil
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflected.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return normalizeUint(reflected.Uint())
	case reflect.Float32, reflect.Float64:
		return normalizeFloat(reflected.Float())
	default:
		return nil, fmt.Errorf("%w: %T", ErrData, value)
	}
}

func normalizeUint(value uint64) (any, error) {
	if value > math.MaxInt64 {
		return nil, fmt.Errorf("%w: %d", ErrData, value)
	}
	return int64(value), nil
}

func normalizeFloat(value float64) (any, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("%w: %v", ErrData, value)
	}
	const exact = 1 << 53
	if value >= -exact && value <= exact && value == math.Trunc(value) {
		return int64(value), nil
	}
	return value, nil
}

func cloneMap(data map[string]any) map[string]any {
	out := maps.Clone(data)
	for key, child := range out {
		out[key] = cloneValue(child)
	}
	return out
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		out := slices.Clone(typed)
		for index := range out {
			out[index] = cloneValue(out[index])
		}
		return out
	default:
		return value
	}
}

func mergeData(left, right map[string]any) (map[string]any, error) {
	out := cloneMap(left)
	if out == nil {
		out = map[string]any{}
	}
	for key, rightValue := range right {
		leftValue, exists := out[key]
		if !exists {
			out[key] = cloneValue(rightValue)
			continue
		}
		merged, err := mergeValue(leftValue, rightValue)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = merged
	}
	return out, nil
}

func mergeValue(left, right any) (any, error) {
	if reflect.DeepEqual(left, right) {
		return cloneValue(left), nil
	}
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	if leftIsMap && rightIsMap {
		return mergeData(leftMap, rightMap)
	}
	return nil, ErrData
}
