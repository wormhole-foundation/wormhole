package fixture

func blockingSend() {
	c := make(chan int, 1)
	c <- 1
}
