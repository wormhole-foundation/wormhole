package devnet

import (
	"errors"
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

// GetFirstGuardianNameFromBootstrapPeers extracts the hostname of the first peer from the bootstrap peers string.
func GetFirstGuardianNameFromBootstrapPeers(bootstrapPeers string) (string, error) {
	peers := strings.Split(bootstrapPeers, ",")
	if len(peers) == 0 {
		return "", errors.New("failed to parse devnet bootstrap peers")
	}
	fields := strings.Split(peers[0], "/")
	if len(fields) < 3 {
		return "", errors.New("failed to parse devnet first bootstrap peer")
	}
	return fields[2], nil
}
