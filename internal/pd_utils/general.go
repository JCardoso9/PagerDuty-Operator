package pd_utils

func Ptr[T any](v T) *T {
	return &v
}
