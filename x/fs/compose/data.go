package compose

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
)

func (file File) normalized() (File, error) {
	out := File{Type: file.Type, Mode: file.Mode}
	if len(file.Values) > 0 {
		out.Values = make(map[string]Slot, len(file.Values))
		for key, slot := range file.Values {
			out.Values[key] = slot
		}
	}
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
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		return normalizeUint(uint64(typed))
	case uint8:
		return normalizeUint(uint64(typed))
	case uint16:
		return normalizeUint(uint64(typed))
	case uint32:
		return normalizeUint(uint64(typed))
	case uint64:
		return normalizeUint(typed)
	case float32:
		return normalizeFloat(float64(typed))
	case float64:
		return normalizeFloat(typed)
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

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = cloneValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = cloneValue(child)
		}
		return out
	default:
		return value
	}
}

func mergeData(left, right map[string]any) (map[string]any, error) {
	out := map[string]any{}
	if left != nil {
		cloned, ok := cloneValue(left).(map[string]any)
		if !ok {
			return nil, ErrData
		}
		out = cloned
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
