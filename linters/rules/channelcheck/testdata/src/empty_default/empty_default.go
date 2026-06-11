package fixture

// Match - empty default case
func emptyDefault() {
	c := make(chan int, 1)
	select {
	case c <- 1: // want `empty default case`
	default:
	}
}
