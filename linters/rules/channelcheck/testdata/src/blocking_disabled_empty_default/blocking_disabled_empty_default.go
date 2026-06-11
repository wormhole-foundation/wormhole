package fixture

func emptyDefault() {
	c := make(chan int, 1)
	select {
	case c <- 1:
	default:
	}
}
