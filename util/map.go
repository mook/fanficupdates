package util

func Map[F any, T any](x []F, mapper func(input F) T) []T {
	var results []T
	for _, e := range x {
		results = append(results, mapper(e))
	}
	return results
}
