package stacks

import (
	"crypto/sha256"
	"fmt"
	"slices"
)

const C32_CHARACTERS = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func doubleSha256Checksum(data []byte) []byte {
	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:4]
}

func c32Encode(src []byte) string {
	size := (len(src)*8 + 4) / 5
	result := make([]byte, 0, size)
	carry := uint16(0)
	carryBits := uint8(0)

	for i := len(src) - 1; i >= 0; i-- {
		currentValue := uint16(src[i])
		lowBitsToTake := 5 - carryBits
		lowBits := currentValue & ((1 << lowBitsToTake) - 1)
		c32Value := (lowBits << carryBits) + carry
		result = append(result, C32_CHARACTERS[c32Value])
		carryBits = (8 + carryBits) - 5
		carry = currentValue >> (8 - carryBits)

		if carryBits >= 5 {
			c32Value = carry & ((1 << 5) - 1)
			result = append(result, C32_CHARACTERS[c32Value])
			carryBits -= 5
			carry >>= 5
		}
	}

	if carryBits > 0 {
		result = append(result, C32_CHARACTERS[carry])
	}

	// Remove leading zeros from c32 encoding
	for len(result) > 0 && result[len(result)-1] == C32_CHARACTERS[0] {
		result = result[:len(result)-1]
	}

	// Add leading zeros from input
	for _, currentValue := range src {
		if currentValue == 0 {
			result = append(result, C32_CHARACTERS[0])
		} else {
			break
		}
	}

	// Reverse the result
	slices.Reverse(result)

	return string(result)
}

func c32CheckEncode(version uint8, data []byte) (string, error) {
	if version >= 32 {
		return "", fmt.Errorf("invalid version %x", version)
	}

	checkData := append([]byte{version}, data...)
	checksum := doubleSha256Checksum(checkData)

	encodingData := append(data, checksum...)

	c32String := string(c32Encode(encodingData))
	versionChar := C32_CHARACTERS[version]

	return string(versionChar) + c32String, nil
}

func stacksAddressEncode(version uint8, data []byte) (string, error) {
	c32CheckEncoded, err := c32CheckEncode(version, data)
	if err != nil {
		return "", err
	}
	return "S" + c32CheckEncoded, nil
}
