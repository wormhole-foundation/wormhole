package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	// Generate 128-bit key
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	// Log key as base64 string
	log.Printf("key: %s", base64.StdEncoding.EncodeToString(key))

	// Encrypt
	ciphertext, err := common.EncryptAESGCM(in, key)
	if err != nil {
		log.Fatalf("failed to encrypt: %v", err)
	}

	// Convert ciphertext as base64 string.
	b64 := base64.StdEncoding.EncodeToString(ciphertext)

	// Hard-wrap to 80 characters per line.
	for i := 0; i < len(b64); i += 80 {
		j := i + 80
		if j > len(b64) {
			j = len(b64)
		}
		fmt.Println(b64[i:j])
	}
}
