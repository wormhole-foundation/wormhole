package fixture

type holder struct {
	ignoreMe chan int
}

// ignoreMe and h.ignoreMe are not matched based on ignore channel rules.
// tracked is still matched.
func ignoreByName() {
	ignoreMe := make(chan int, 1)
	tracked := make(chan int, 1)
	h := &holder{ignoreMe: make(chan int, 1)}

	ignoreMe <- 1   // ignored by name
	h.ignoreMe <- 2 // ignored by selector field name
	tracked <- 3    // want `Blocking send`
}
