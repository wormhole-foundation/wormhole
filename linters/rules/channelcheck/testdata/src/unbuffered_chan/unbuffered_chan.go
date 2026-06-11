package fixture

func unbuffered() {
	_ = make(chan int) // want `unbuffered channel`
}
