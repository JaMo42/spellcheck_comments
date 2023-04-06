// Package util contains common utility functions. This is not part of the common
// package as that is imported without namespacing.
package util

// Contains returns trues if arr contains value.
func Contains[T comparable](arr []T, value T) bool {
	_, exists := Position(arr, value)
	return exists
}

// Position returns the position of value in arr and if arr contains value.
// If arr does not contain value the index one past the last element is returned.
func Position[T comparable](arr []T, value T) (int, bool) {
	for idx, elem := range arr {
		if elem == value {
			return idx, true
		}
	}
	return len(arr), false
}

// Filter retains all values of arr for which pred returns true.
func Filter[T any](arr []T, pred func(T) bool) []T {
	last := len(arr) - 1
	for i := last; i >= 0; i -= 1 {
		if !pred(arr[i]) {
			arr[i] = arr[last]
			last -= 1
		}
	}
	return arr[:last+1]
}

// Map applies f to each value of arr.
func Map[T any, U any](arr []T, f func(T) U) []U {
	result := make([]U, len(arr))
	for i, x := range arr {
		result[i] = f(x)
	}
	return result
}

// String2Int creates an integer with the utf-8 representation of the string.
// The last character is the lowest byte.
func String2Int(s string) uint64 {
	asInt := uint64(0)
	for _, b := range []byte(s) {
		asInt <<= 8
		asInt |= uint64(b)
	}
	return asInt
}

// Xxs pops the first value from the given slice.
func Xxs[T any](arr []T) (T, []T) {
	x, xs := arr[0], arr[1:]
	return x, xs
}
