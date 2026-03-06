package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

func DecryptAESGCM(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %v", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data is too short")
	}
	nonce, data := data[:nonceSize], data[nonceSize:]
	out, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}
	return out, nil
}

func EncryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %v", err)
	}
	// Preallocate the output to avoid an extra allocation in append(nonce, out...).
	out := make([]byte, gcm.NonceSize(), gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	nonce := out[:gcm.NonceSize()]
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to read random data: %v", err)
	}
	out = gcm.Seal(out, nonce, plaintext, nil)
	return out, nil
}
