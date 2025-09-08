package app

func InSlice[T comparable](x T, list []T) bool {
	for _, e := range list {
		if e == x {
			return true
		}
	}
	return false
}
