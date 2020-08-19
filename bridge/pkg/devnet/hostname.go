package devnet

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetDevnetIndex returns the current host's devnet index (i.e. 0 for guardian-0).
func GetDevnetIndex() (int, error) {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	h := strings.Split(hostname, "-")

	if h[0] != "guardian" {
		return 0, fmt.Errorf("hostname %s does not appear to be a devnet host", hostname)
	}

	i, err := strconv.Atoi(h[1])
	if err != nil {
		return 0, fmt.Errorf("invalid devnet index %s in hostname %s", h[1], hostname)
	}

	return i, nil
}
