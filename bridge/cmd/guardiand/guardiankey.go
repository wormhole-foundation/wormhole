package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getDevnetIndex returns the current host's devnet index (i.e. 0 for guardian-0).
func getDevnetIndex() (int, error) {
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

// deterministicKeyByIndex generates a deterministic address from a given index.
func deterministicKeyByIndex(c elliptic.Curve, idx uint64) (*ecdsa.PrivateKey) {
	buf := make([]byte, 200)
	binary.LittleEndian.PutUint64(buf, idx)

	worstRNG := bytes.NewBuffer(buf)

	key, err := ecdsa.GenerateKey(c, bytes.NewReader(worstRNG.Bytes()))
	if err != nil {
		panic(err)
	}

	return key
}

