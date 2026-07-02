package fixture

import "time"

// A send inside a case BODY is unrelated to the select's escape mechanism —
// it executes synchronously when that branch is chosen and blocks just like
// any other bare send. Even with a real timer escape on the select, the rule
// must still flag the in-body sends.
func sendInCaseBody() {
	c := make(chan int, 1)
	other := make(chan int, 1)
	select {
	case <-time.After(time.Second):
		c <- 1 // want `Blocking send`
	case <-other:
		c <- 2 // want `Blocking send`
	}
}
