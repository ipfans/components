package utils

func DefaultValue[T comparable](value T, defaults ...T) T {
	// 根据是否为元素默认值，是则返回 defaults[0]，否则返回 value
	var zero T
	if value == zero {
		return defaults[0]
	}
	return value
}
