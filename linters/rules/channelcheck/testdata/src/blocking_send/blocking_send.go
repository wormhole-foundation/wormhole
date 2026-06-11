package fixture

// Single match
func blockingSend() {
	c := make(chan int, 1)
	c <- 1 // want `Blocking send`
}
