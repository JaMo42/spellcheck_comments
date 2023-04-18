// Package util contains common utility functions. This is not part of the common
// package as that is imported without namespacing.
package util

import (
	"github.com/mattn/go-runewidth"
)

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

// Sum returns the sum of the elements in arr.
func Sum(arr []int) int {
	sum := 0
	for _, x := range arr {
		sum += x
	}
	return sum
}

// PopFront pops the first value from the given slice.
func PopFront[T any](arr []T) (T, []T) {
	x, xs := arr[0], arr[1:]
	return x, xs
}

// Back returns a pointer to the last element in the slice.
func Back[T any](arr []T) *T {
	return &arr[len(arr)-1]
}

// MaxElem returns the maximum element of an integer slice.
func MaxElem(arr []int) (max int) {
	for _, i := range arr {
		if i > max {
			max = i
		}
	}
	return max
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

// StrLen simply wraps a call to len(s). Unlike len this can be used as a
// parameter to other functions.
func StrLen(s string) int {
	return len(s)
}

// CeilDiv performs ceiling integer division of a / b.
func CeilDiv(a, b int) int {
	return (a + b - 1) / b
}

// Min returns the minimum of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp clamps the value in the given inclusive range.
func Clamp(x, lo, hi int) int {
	return Min(Max(x, lo), hi)
}

// FixPrintfPadding returns the correct padding amount to pad the given word
// by the wanted number of cells, handling codepoints with multiple bytes and
// fullwidth characters.
func FixPrintfPadding(str string, padding int) int {
	byteCount := len(str)
	width := runewidth.StringWidth(str)
	return padding - byteCount + width
}

func Deduplicate[T comparable](arr []T) []T {
	set := make(map[T]bool)
	for _, x := range arr {
		set[x] = true
	}
	filtered := make([]T, len(set))
	i := 0
	for x := range set {
		filtered[i] = x
		i++
	}
	return filtered
}
