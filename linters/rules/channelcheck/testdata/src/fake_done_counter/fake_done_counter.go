package fixture

// Non-match. Only the ctx.Done() function should trigger this.
type fakeDone struct {
	done chan struct{}
}

func (f *fakeDone) Done() <-chan struct{} {
	return f.done
}

func fakeDoneCounter() {
	c := make(chan int, 1)
	f := &fakeDone{done: make(chan struct{}, 1)}
	select {
	case c <- 1: // want `Blocking send`
	case <-f.Done():
	}
}
