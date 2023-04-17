package ibc

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/tidwall/gjson"

	"go.uber.org/zap"
)

// WasmAttributes is an object to facilitate parsing of wasm event attributes. It provides a method to parse the attribute array in a wasm event,
// plus methods to return attributes as the appropriate type, including doing range checking.
type WasmAttributes struct {
	m map[string]string
}

// GetAsString returns the attribute value as a string.
func (wa *WasmAttributes) GetAsString(key string) (string, error) {
	value, exists := wa.m[key]
	if !exists {
		return "", fmt.Errorf("attribute %s does not exist", key)
	}

	return value, nil
}

// GetAsUint returns the attribute value as an unsigned int. It also performs range checking.
func (wa *WasmAttributes) GetAsUint(key string, bitSize int) (uint64, error) {
	valueStr, exists := wa.m[key]
	if !exists {
		return 0, fmt.Errorf("attribute %s does not exist", key)
	}

	value, err := strconv.ParseUint(valueStr, 10, bitSize)
	if err != nil {
		return 0, fmt.Errorf("failed parse attribute %s with value %s as %d bit uint: %w", key, valueStr, bitSize, err)
	}

	return value, nil
}

// GetAsInt returns the attribute value as a signed int. It also performs range checking.
func (wa *WasmAttributes) GetAsInt(key string, bitSize int) (int64, error) {
	valueStr, exists := wa.m[key]
	if !exists {
		return 0, fmt.Errorf("attribute %s does not exist", key)
	}

	value, err := strconv.ParseInt(valueStr, 10, bitSize)
	if err != nil {
		return 0, fmt.Errorf("failed parse attribute %s with value %s as %d bit int: %w", key, valueStr, bitSize, err)
	}

	return value, nil
}

// Parse parses the attributes in a wasm event.
func (wa *WasmAttributes) Parse(logger *zap.Logger, event gjson.Result) error {
	wa.m = make(map[string]string)
	attributes := gjson.Get(event.String(), "attributes")
	if !attributes.Exists() {
		return fmt.Errorf("event does not contain any attributes")
	}

	for _, attribute := range attributes.Array() {
		if !attribute.IsObject() {
			return fmt.Errorf("event attribute is invalid: %s", attribute.String())
		}
		keyBase := gjson.Get(attribute.String(), "key")
		if !keyBase.Exists() {
			return fmt.Errorf("event attribute does not have a key: %s", attribute.String())
		}
		valueBase := gjson.Get(attribute.String(), "value")
		if !valueBase.Exists() {
			return fmt.Errorf("event attribute does not have a value: %s", attribute.String())
		}
		keyRaw, err := base64.StdEncoding.DecodeString(keyBase.String())
		if err != nil {
			return fmt.Errorf("event attribute key is invalid base64: %s", attribute.String())
		}
		valueRaw, err := base64.StdEncoding.DecodeString(valueBase.String())
		if err != nil {
			return fmt.Errorf("event attribute value is invalid base64: %s", attribute.String())
		}

		key := string(keyRaw)
		value := string(valueRaw)

		if _, ok := wa.m[key]; ok {
			return fmt.Errorf("duplicate key in event: %s", key)
		}

		logger.Debug("event attribute", zap.String("key", key), zap.String("value", value))
		wa.m[key] = value
	}

	return nil
}
