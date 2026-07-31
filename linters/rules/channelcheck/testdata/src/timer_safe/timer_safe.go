// OK
package fixture

import "time"

func timerSafe() {
	c := make(chan int, 1)
	select {
	case c <- 1:
	case <-time.After(time.Second):
	}
}
