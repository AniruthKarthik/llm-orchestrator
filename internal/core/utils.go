package core

// DeepCopyValue recursively copies any value, specifically handling maps and slices.
func DeepCopyValue(v any) any {
	switch v := v.(type) {
	case map[string]any:
		return DeepCopyMap(v)
	case []any:
		return DeepCopySlice(v)
	case []string:
		return DeepCopyStringSlice(v)
	case []int:
		cp := make([]int, len(v))
		copy(cp, v)
		return cp
	case []float64:
		cp := make([]float64, len(v))
		copy(cp, v)
		return cp
	case []bool:
		cp := make([]bool, len(v))
		copy(cp, v)
		return cp
	default:
		// Basic types (int, string, bool, time.Time, etc.) are passed by value in Go.
		return v
	}
}

// DeepCopyMap recursively copies a map[string]any.
func DeepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = DeepCopyValue(v)
	}
	return cp
}

// DeepCopySlice recursively copies a slice of any.
func DeepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	cp := make([]any, len(s))
	for i, v := range s {
		cp[i] = DeepCopyValue(v)
	}
	return cp
}

// DeepCopyStringSlice copies a slice of strings.
func DeepCopyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}
