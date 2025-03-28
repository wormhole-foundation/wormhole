package query

func makeChannelPair[T any](capacity int) (<-chan T, chan<- T) {
	out := make(chan T, capacity)
	return out, out
}
