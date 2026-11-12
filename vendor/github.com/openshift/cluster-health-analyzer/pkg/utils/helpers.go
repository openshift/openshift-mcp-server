package utils

// Ptr is an helper function to return the pointer of any type
func Ptr[T any](v T) *T {
	return &v
}
