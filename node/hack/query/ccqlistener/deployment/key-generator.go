package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/openpgp/armor"
)

const (
	CCQ_SERVER_SIGNING_KEY = "CCQ SERVER SIGNING KEY"
)

// createGuardianKeyProtobuf manually creates a binary compatible with the
// Wormhole GuardianKey protobuf format:
// message GuardianKey {
//   bytes Data = 1;
//   bool UnsafeDeterministicKey = 2;
// }
func createGuardianKeyProtobuf(privateKeyBytes []byte) []byte {
	var buf bytes.Buffer

	// Wire format for bytes field: tag (field_number << 3 | wire_type)
	// Field 1, wire type 2 (length-delimited) = 0x0A
	buf.WriteByte(0x0A)
	
	// Write length of private key bytes as varint
	buf.WriteByte(byte(len(privateKeyBytes)))
	
	// Write private key bytes
	buf.Write(privateKeyBytes)
	
	// Wire format for bool field: tag | value
	// Field 2, wire type 0 (varint) = 0x10, value 0 (false) = 0
	buf.WriteByte(0x10)
	buf.WriteByte(0x00) // false for UnsafeDeterministicKey
	
	return buf.Bytes()
}

func main() {
	// Create the keys directory if it doesn't exist
	err := os.MkdirAll("/app/keys", 0755)
	if err != nil {
		log.Fatalf("Failed to create keys directory: %v", err)
	}

	keyPath := filepath.Join("/app/keys", "ccqlistener.signerKey")
	
	// Check if the key already exists
	if _, err := os.Stat(keyPath); err == nil {
		log.Printf("Key file already exists at %s. Not overwriting.", keyPath)
		return
	}

	// Generate a new ECDSA private key using the secp256k1 curve (Ethereum's curve)
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// Get raw private key bytes
	privateKeyBytes := crypto.FromECDSA(privateKey)
	
	// Manually create protobuf binary format
	protobufBytes := createGuardianKeyProtobuf(privateKeyBytes)

	// Open file for writing
	f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// Get the Ethereum address for the public key
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create armor headers
	headers := map[string]string{
		"PublicKey": address.Hex(),
	}

	// Create the armored writer
	armorWriter, err := armor.Encode(f, CCQ_SERVER_SIGNING_KEY, headers)
	if err != nil {
		log.Fatalf("Failed to create armor encoder: %v", err)
	}
	defer armorWriter.Close()

	// Write the protobuf bytes directly
	_, err = armorWriter.Write(protobufBytes)
	if err != nil {
		log.Fatalf("Failed to write key: %v", err)
	}

	log.Printf("Generated new key at %s", keyPath)
	log.Printf("Public Key (add to ccqAllowedRequesters): %s", address.Hex())
} 