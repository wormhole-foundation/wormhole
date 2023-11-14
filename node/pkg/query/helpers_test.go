package query

func makeChannelPair[T any](cap int) (<-chan T, chan<- T) {
	out := make(chan T, cap)
	return out, out
}
