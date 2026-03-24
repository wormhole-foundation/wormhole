// Script to generate a delegated manager set from devnet guardian keys.
// Usage: go run main.go
package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
)

// First 7 devnet guardian private keys from scripts/devnet-consts.json
var devnetGuardianPrivateKeys = []string{
	"cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0", // guardian-0
	"c3b2e45c422a1602333a64078aeb42637370b0f48fe385f9cfa6ad54a8e0c47e", // guardian-1
	"9f790d3f08bc4b5cd910d4278f3deb406e57bb5e924906ccd52052bb078ccd47", // guardian-2
	"b20cc49d6f2c82a5e6519015fc18aa3e562867f85f872c58f1277cfbd2a0c8e4", // guardian-3
	"eded5a2fdcb5bbbfa5b07f2a91393813420e7ac30a72fc935b6df36f8294b855", // guardian-4
	"00d39587c3556f289677a837c7f3c0817cb7541ce6e38a243a4bdc761d534c5e", // guardian-5
	"da534d61a8da77b232f3a2cee55c0125e2b3e33a5cd8247f3fe9e72379445c3b", // guardian-6
}

func main() {
	fmt.Println("// KnownDevnetManagerSet is the initial delegated manager set for devnet.")
	fmt.Println("// It is a 5-of-7 multisig using the first 7 devnet guardian keys.")
	fmt.Println("var KnownDevnetManagerSet = struct {")
	fmt.Println("\tM          uint8")
	fmt.Println("\tN          uint8")
	fmt.Println("\tPublicKeys [][33]byte")
	fmt.Println("}{")
	fmt.Println("\tM: 5,")
	fmt.Println("\tN: 7,")
	fmt.Println("\tPublicKeys: [][33]byte{")

	for i, privKeyHex := range devnetGuardianPrivateKeys {
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		if err != nil {
			log.Fatalf("Failed to decode private key %d: %v", i, err)
		}

		// Parse as ECDSA private key
		privKey, err := crypto.ToECDSA(privKeyBytes)
		if err != nil {
			log.Fatalf("Failed to parse private key %d: %v", i, err)
		}

		// Get compressed public key (33 bytes)
		compressedPubKey := compressPublicKey(&privKey.PublicKey)

		// Format as Go byte array
		fmt.Printf("\t\t// guardian-%d\n", i)
		fmt.Printf("\t\t{")
		for j, b := range compressedPubKey {
			if j > 0 {
				fmt.Printf(", ")
			}
			if j > 0 && j%16 == 0 {
				fmt.Printf("\n\t\t\t")
			}
			fmt.Printf("0x%02X", b)
		}
		fmt.Printf("},\n")
	}

	fmt.Println("\t},")
	fmt.Println("}")

	// Also print as hex strings for reference
	fmt.Println("\n// Hex representation for reference:")
	for i, privKeyHex := range devnetGuardianPrivateKeys {
		privKeyBytes, _ := hex.DecodeString(privKeyHex)
		privKey, _ := crypto.ToECDSA(privKeyBytes)
		compressedPubKey := compressPublicKey(&privKey.PublicKey)
		fmt.Printf("// guardian-%d: %s\n", i, hex.EncodeToString(compressedPubKey))
	}
}

// compressPublicKey converts an ECDSA public key to compressed secp256k1 format (33 bytes).
func compressPublicKey(pubKey *ecdsa.PublicKey) []byte {
	// Pad X and Y to 32 bytes each
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Ensure 32 bytes
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	// Create uncompressed pubkey: 0x04 || X || Y
	uncompressed := make([]byte, 65)
	uncompressed[0] = 0x04
	copy(uncompressed[1:33], xPadded)
	copy(uncompressed[33:65], yPadded)

	btcPubKey, err := btcec.ParsePubKey(uncompressed)
	if err != nil {
		log.Fatalf("Failed to parse pubkey: %v", err)
	}
	return btcPubKey.SerializeCompressed()
}
