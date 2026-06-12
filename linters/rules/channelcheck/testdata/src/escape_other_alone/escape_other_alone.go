package fixture

// Match for no default or timeout
func escapeOtherAlone() {
	c := make(chan int, 1)
	other := make(chan struct{}, 1)
	select {
	case c <- 1: // want `Blocking send`
	case <-other:
	}
}
