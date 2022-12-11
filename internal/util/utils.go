package util

func Map[T any, K any](vs []T, f func(T) K) []K {
	vsm := make([]K, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
