// OK
package fixture

import "fmt"

// Channel send with default and a println. This is "safe" according to the code.
func defaultSafe() {
	c := make(chan int, 1)
	select {
	case c <- 1:
	default:
		fmt.Println("dropped")
	}
}
