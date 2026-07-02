package fixture

// Single match for buffer too large
func tooLarge() {
	_ = make(chan int, 100) // want `buffer size exceeds`
}
