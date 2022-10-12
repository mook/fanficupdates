package util

// Map an input slice to an output slice with the given mapper function.
func Map[F any, T any](slice []F, mapper func(input F) T) []T {
	results := make([]T, 0, len(slice))
	for _, e := range slice {
		results = append(results, mapper(e))
	}
	return results
}

// Any checks if any element of input slice causes the predicate to return true.
func Any[T any](slice []T, predicate func(input T) bool) bool {
	for _, e := range slice {
		if predicate(e) {
			return true
		}
	}
	return false
}

// Find the first element in the given slice that causes the predicate to
// return true; if not found, returns nil.
func Find[T any](slice []T, predicate func(input T) bool) *T {
	for _, e := range slice {
		if predicate(e) {
			return &e
		}
	}
	return nil
}

// Filter the given slice, keeping only the ones that cause the predicate to
// return true.
func Filter[T any](slice []T, predicate func(input T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, e := range slice {
		if predicate(e) {
			result = append(result, e)
		}
	}
	return result
}
