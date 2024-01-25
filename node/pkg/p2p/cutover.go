package p2p

import (
	"strings"
)

// cutOverBootstrapPeers updates the bootstrap peers to reflect the new quic-v1. It assumes that the string has previously been validated.
func cutOverBootstrapPeers(bootstrapPeers string) string {
	return strings.ReplaceAll(bootstrapPeers, "/quic/", "/quic-v1/")
}

// cutOverAddressPattern updates the address patterns. It assumes that the string is valid.
func cutOverAddressPattern(pattern string) string {
	if !strings.Contains(pattern, "/quic-v1") {
		// These patterns are hardcoded so we are not worried about invalid values.
		pattern = strings.ReplaceAll(pattern, "/quic", "/quic-v1")
	}

	return pattern
}
