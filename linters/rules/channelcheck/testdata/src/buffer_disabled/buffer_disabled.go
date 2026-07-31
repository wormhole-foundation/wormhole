package fixture

func tooLarge() {
	_ = make(chan int, 100)
}
