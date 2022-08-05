package collectionutil

func Contains[T comparable](elems []T, v T) bool {
	for _, e := range elems {
		if v == e {
			return true
		}
	}
	return false
}
