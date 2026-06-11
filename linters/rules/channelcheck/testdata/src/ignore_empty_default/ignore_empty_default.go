package fixture

func ignoreEmptyDefault() {
	ignoreMe := make(chan int, 1)
	select {
	case ignoreMe <- 1:
	default:
	}
}
