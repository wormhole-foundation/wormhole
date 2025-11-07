package stacks

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"testing"
)

// Helper function to create a bytes.Reader from hex string
func hexReader(hexStr string) *bytes.Reader {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(data)
}

// Helper function to encode a uint32 as big-endian hex
func uint32Hex(val uint32) string {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, val)
	return hex.EncodeToString(buf)
}

func TestInt128Signed_Positive(t *testing.T) {
	// Type 0x00 + 16 bytes for value 42
	reader := hexReader("00" + "0000000000000000000000000000002a")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Int128: %v", err)
	}
	int128, ok := val.(*Int128)
	if !ok {
		t.Fatalf("Expected *Int128, got %T", val)
	}
	expected := big.NewInt(42)
	if int128.Value.Cmp(expected) != 0 {
		t.Errorf("Expected %s, got %s", expected.String(), int128.Value.String())
	}
}

func TestInt128Signed_Negative(t *testing.T) {
	// Type 0x00 + 16 bytes for value -1 (all FFs in two's complement)
	reader := hexReader("00" + "ffffffffffffffffffffffffffffffff")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Int128: %v", err)
	}
	int128, ok := val.(*Int128)
	if !ok {
		t.Fatalf("Expected *Int128, got %T", val)
	}
	expected := big.NewInt(-1)
	if int128.Value.Cmp(expected) != 0 {
		t.Errorf("Expected %s, got %s", expected.String(), int128.Value.String())
	}
}

func TestInt128Signed_NegativeMissingChars(t *testing.T) {
	// Type 0x00 + 15 bytes for value -1 (all FFs in two's complement)
	reader := hexReader("00" + "ffffffffffffffffffffffffffffff")
	_, err := DecodeClarityValue(reader)
	if err == nil || !contains(err.Error(), "unexpected EOF") {
		t.Fatalf("Failed to decode buffer EOF: %v", err)
	}
}

// Regression test from previous bug
func TestInt128Signed_NegativeLarge(t *testing.T) {
	// Type 0x00 + 16 bytes for value -42
	// Two's complement of 42: 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFD6
	reader := hexReader("00" + "ffffffffffffffffffffffffffffffd6")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Int128: %v", err)
	}
	int128, ok := val.(*Int128)
	if !ok {
		t.Fatalf("Expected *Int128, got %T", val)
	}
	expected := big.NewInt(-42)
	if int128.Value.Cmp(expected) != 0 {
		t.Errorf("Expected %s, got %s", expected.String(), int128.Value.String())
	}
}

func TestUInt128_Positive(t *testing.T) {
	// Type 0x01 + 16 bytes for value 12345
	reader := hexReader("01" + "00000000000000000000000000003039")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode UInt128: %v", err)
	}
	uint128, ok := val.(*UInt128)
	if !ok {
		t.Fatalf("Expected *UInt128, got %T", val)
	}
	expected := big.NewInt(12345)
	if uint128.Value.Cmp(expected) != 0 {
		t.Errorf("Expected %s, got %s", expected.String(), uint128.Value.String())
	}
}

func TestBuffer_Valid(t *testing.T) {
	// Type 0x02 + length 5 + "hello"
	data := "hello"
	reader := hexReader("02" + uint32Hex(5) + hex.EncodeToString([]byte(data)))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Buffer: %v", err)
	}
	buffer, ok := val.(*ClarityBuffer)
	if !ok {
		t.Fatalf("Expected *ClarityBuffer, got %T", val)
	}
	if buffer.Length != 5 {
		t.Errorf("Expected length 5, got %d", buffer.Length)
	}
	if string(buffer.Data) != data {
		t.Errorf("Expected %s, got %s", data, string(buffer.Data))
	}
}

func TestBuffer_ExceedsLimit(t *testing.T) {
	// Type 0x02 + length exceeding MaxClarityBufferLength
	oversizeLength := uint32(MaxClarityBufferLength + 1)
	reader := hexReader("02" + uint32Hex(oversizeLength))
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for oversized buffer, got nil")
	}
	expectedErr := "buffer length"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestBooleanTrue(t *testing.T) {
	// Type 0x03
	reader := hexReader("03")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Boolean: %v", err)
	}
	boolean, ok := val.(*Bool)
	if !ok {
		t.Fatalf("Expected *Bool, got %T", val)
	}
	if !boolean.Value {
		t.Error("Expected true, got false")
	}
}

func TestBooleanFalse(t *testing.T) {
	// Type 0x04
	reader := hexReader("04")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Boolean: %v", err)
	}
	boolean, ok := val.(*Bool)
	if !ok {
		t.Fatalf("Expected *Bool, got %T", val)
	}
	if boolean.Value {
		t.Error("Expected false, got true")
	}
}

func TestStandardPrincipal(t *testing.T) {
	// Type 0x05 + version 0x16 + 20 bytes hash160
	hash := "0102030405060708090a0b0c0d0e0f1011121314"
	reader := hexReader("05" + "16" + hash)
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode StandardPrincipal: %v", err)
	}
	principal, ok := val.(*ClarityPrincipal)
	if !ok {
		t.Fatalf("Expected *ClarityPrincipal, got %T", val)
	}
	if principal.Version != 0x16 {
		t.Errorf("Expected version 0x16, got 0x%02x", principal.Version)
	}
	expectedHash, _ := hex.DecodeString(hash)
	if !bytes.Equal(principal.Hash160[:], expectedHash) {
		t.Errorf("Hash160 mismatch")
	}
}

func TestContractPrincipal(t *testing.T) {
	// Type 0x06 + version 0x16 + 20 bytes hash160 + name length + name
	hash := "0102030405060708090a0b0c0d0e0f1011121314"
	contractName := "mycontract"
	nameLen := byte(len(contractName))
	reader := hexReader("06" + "16" + hash + hex.EncodeToString([]byte{nameLen}) + hex.EncodeToString([]byte(contractName)))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode ContractPrincipal: %v", err)
	}
	contractPrincipal, ok := val.(*ClarityContractPrincipal)
	if !ok {
		t.Fatalf("Expected *ClarityContractPrincipal, got %T", val)
	}
	if contractPrincipal.Name != contractName {
		t.Errorf("Expected name %s, got %s", contractName, contractPrincipal.Name)
	}
}

func TestOkResponse(t *testing.T) {
	// Type 0x07 + nested boolean true (0x03)
	reader := hexReader("07" + "03")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Response: %v", err)
	}
	response, ok := val.(*Response)
	if !ok {
		t.Fatalf("Expected *Response, got %T", val)
	}
	if !response.IsOk {
		t.Error("Expected IsOk=true, got false")
	}
	innerBool, ok := response.Result.(*Bool)
	if !ok || !innerBool.Value {
		t.Error("Expected inner value to be true")
	}
}

func TestErrResponse(t *testing.T) {
	// Type 0x08 + nested boolean false (0x04)
	reader := hexReader("08" + "04")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Response: %v", err)
	}
	response, ok := val.(*Response)
	if !ok {
		t.Fatalf("Expected *Response, got %T", val)
	}
	if response.IsOk {
		t.Error("Expected IsOk=false, got true")
	}
	innerBool, ok := response.Result.(*Bool)
	if !ok || innerBool.Value {
		t.Error("Expected inner value to be false")
	}
}

// Regression test from previous bug.
func TestNoneOption(t *testing.T) {
	// Type 0x09 - should NOT read any additional data
	reader := hexReader("09")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Option: %v", err)
	}
	option, ok := val.(*Option)
	if !ok {
		t.Fatalf("Expected *Option, got %T", val)
	}
	if option.IsSome {
		t.Error("Expected IsSome=false, got true")
	}
	if option.Value != nil {
		t.Error("Expected Value=nil for None")
	}
	// Verify no bytes were consumed beyond type ID
	if reader.Len() != 0 {
		t.Errorf("Expected 0 bytes remaining, got %d", reader.Len())
	}
}

func TestSomeOption(t *testing.T) {
	// Type 0x0a + nested boolean true (0x03)
	reader := hexReader("0a" + "03")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Option: %v", err)
	}
	option, ok := val.(*Option)
	if !ok {
		t.Fatalf("Expected *Option, got %T", val)
	}
	if !option.IsSome {
		t.Error("Expected IsSome=true, got false")
	}
	innerBool, ok := option.Value.(*Bool)
	if !ok || !innerBool.Value {
		t.Error("Expected inner value to be true")
	}
}

func TestList_Valid(t *testing.T) {
	// Type 0x0b + length 3 + three booleans (true, false, true)
	reader := hexReader("0b" + uint32Hex(3) + "03" + "04" + "03")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode List: %v", err)
	}
	list, ok := val.(*List)
	if !ok {
		t.Fatalf("Expected *List, got %T", val)
	}
	if list.Length != 3 {
		t.Errorf("Expected length 3, got %d", list.Length)
	}
	if len(list.Values) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(list.Values))
	}
	// Check first element is true
	b0, ok := list.Values[0].(*Bool)
	if !ok || !b0.Value {
		t.Error("Expected first element to be true")
	}
	// Check second element is false
	b1, ok := list.Values[1].(*Bool)
	if !ok || b1.Value {
		t.Error("Expected second element to be false")
	}
	// Check third element is true
	b2, ok := list.Values[2].(*Bool)
	if !ok || !b2.Value {
		t.Error("Expected third element to be true")
	}
}

func TestList_ExceedsLimit(t *testing.T) {
	// Type 0x0b + length exceeding MaxClarityListLength
	oversizeLength := uint32(MaxClarityListLength + 1)
	reader := hexReader("0b" + uint32Hex(oversizeLength))
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for oversized list, got nil")
	}
	expectedErr := "list length"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestTuple_Valid(t *testing.T) {
	// Type 0x0c + length 2 + ("foo": true, "bar": false)
	// Field 1: name length 3 + "foo" + boolean true (0x03)
	// Field 2: name length 3 + "bar" + boolean false (0x04)
	fooHex := hex.EncodeToString([]byte("foo"))
	barHex := hex.EncodeToString([]byte("bar"))
	reader := hexReader("0c" + uint32Hex(2) +
		"03" + fooHex + "03" +
		"03" + barHex + "04")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode Tuple: %v", err)
	}
	tuple, ok := val.(*Tuple)
	if !ok {
		t.Fatalf("Expected *Tuple, got %T", val)
	}
	if tuple.Length != 2 {
		t.Errorf("Expected length 2, got %d", tuple.Length)
	}
	if len(tuple.Values) != 2 {
		t.Fatalf("Expected 2 values, got %d", len(tuple.Values))
	}
	// Check "foo" field
	fooVal, ok := tuple.Values["foo"].(*Bool)
	if !ok || !fooVal.Value {
		t.Error("Expected 'foo' field to be true")
	}
	// Check "bar" field
	barVal, ok := tuple.Values["bar"].(*Bool)
	if !ok || barVal.Value {
		t.Error("Expected 'bar' field to be false")
	}
}

func TestTuple_ExceedsLimit(t *testing.T) {
	// Type 0x0c + length exceeding MaxClarityTupleLength
	oversizeLength := uint32(MaxClarityTupleLength + 1)
	reader := hexReader("0c" + uint32Hex(oversizeLength))
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for oversized tuple, got nil")
	}
	expectedErr := "tuple length"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestStringASCII_Valid(t *testing.T) {
	// Type 0x0d + length 5 + "hello"
	data := "hello"
	reader := hexReader("0d" + uint32Hex(5) + hex.EncodeToString([]byte(data)))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode StringASCII: %v", err)
	}
	str, ok := val.(*StringASCII)
	if !ok {
		t.Fatalf("Expected *StringASCII, got %T", val)
	}
	if str.Length != 5 {
		t.Errorf("Expected length 5, got %d", str.Length)
	}
	if str.Value != data {
		t.Errorf("Expected %s, got %s", data, str.Value)
	}
}

func TestStringASCII_ExceedsLimit(t *testing.T) {
	// Type 0x0d + length exceeding MaxClarityStringLength
	oversizeLength := uint32(MaxClarityStringLength + 1)
	reader := hexReader("0d" + uint32Hex(oversizeLength))
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for oversized string-ascii, got nil")
	}
	expectedErr := "string-ascii length"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestStringUTF8_Valid(t *testing.T) {
	// Type 0x0e + length 12 + "hello 世界" (UTF-8 encoded)
	data := "hello 世界"
	utf8Bytes := []byte(data)
	//nolint:gosec // Hardcoded constant that isn't larger than uint32.
	reader := hexReader("0e" + uint32Hex(uint32(len(utf8Bytes))) + hex.EncodeToString(utf8Bytes))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode StringUTF8: %v", err)
	}
	str, ok := val.(*StringUTF8)
	if !ok {
		t.Fatalf("Expected *StringUTF8, got %T", val)
	}
	if str.Value != data {
		t.Errorf("Expected %s, got %s", data, str.Value)
	}
}

func TestStringUTF8_ExceedsLimit(t *testing.T) {
	// Type 0x0e + length exceeding MaxClarityStringLength
	oversizeLength := uint32(MaxClarityStringLength + 1)
	reader := hexReader("0e" + uint32Hex(oversizeLength))
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for oversized string-utf8, got nil")
	}
	expectedErr := "string-utf8 length"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestUnknownTypeID(t *testing.T) {
	// Type 0xFF (invalid)
	reader := hexReader("ff")
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for unknown type ID, got nil")
	}
	expectedErr := "unknown clarity type ID"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestNestedStructure(t *testing.T) {
	// Create a complex nested structure:
	// Some(Ok(List[true, false]))
	// Type 0x0a (Some) + 0x07 (Ok) + 0x0b (List) + length 2 + true + false
	reader := hexReader("0a" + "07" + "0b" + uint32Hex(2) + "03" + "04")
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode nested structure: %v", err)
	}

	// Check outer Some
	option, ok := val.(*Option)
	if !ok {
		t.Fatalf("Expected *Option, got %T", val)
	}
	if !option.IsSome {
		t.Fatal("Expected Some")
	}

	// Check inner Ok
	response, ok := option.Value.(*Response)
	if !ok {
		t.Fatalf("Expected *Response, got %T", option.Value)
	}
	if !response.IsOk {
		t.Fatal("Expected Ok")
	}

	// Check List
	list, ok := response.Result.(*List)
	if !ok {
		t.Fatalf("Expected *List, got %T", response.Result)
	}
	if list.Length != 2 {
		t.Fatalf("Expected list length 2, got %d", list.Length)
	}

	// Check list elements
	b0, ok := list.Values[0].(*Bool)
	if !ok || !b0.Value {
		t.Error("Expected first element to be true")
	}
	b1, ok := list.Values[1].(*Bool)
	if !ok || b1.Value {
		t.Error("Expected second element to be false")
	}
}

// Regression test from previous bug.
func TestBuffer_MissingBytes(t *testing.T) {
	// Type 0x02 + length 2 + 01
	reader := hexReader("02" + uint32Hex(2) + "01")
	_, err := DecodeClarityValue(reader)
	if err == nil || !contains(err.Error(), "unexpected EOF") {
		t.Fatalf("Failed to decode buffer EOF: %v", err)
	}
}

// Regression test from previous bug.
func TestBuffer_Empty(t *testing.T) {
	// Type 0x02 + length 0
	reader := hexReader("02" + uint32Hex(0))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode empty buffer: %v", err)
	}
	buffer, ok := val.(*ClarityBuffer)
	if !ok {
		t.Fatalf("Expected *ClarityBuffer, got %T", val)
	}
	if buffer.Length != 0 {
		t.Errorf("Expected length 0, got %d", buffer.Length)
	}
	if len(buffer.Data) != 0 {
		t.Errorf("Expected empty data, got %d bytes", len(buffer.Data))
	}
}

func TestList_Empty(t *testing.T) {
	// Type 0x0b + length 0
	reader := hexReader("0b" + uint32Hex(0))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode empty list: %v", err)
	}
	list, ok := val.(*List)
	if !ok {
		t.Fatalf("Expected *List, got %T", val)
	}
	if list.Length != 0 {
		t.Errorf("Expected length 0, got %d", list.Length)
	}
	if len(list.Values) != 0 {
		t.Errorf("Expected empty values, got %d", len(list.Values))
	}
}

func TestTuple_Empty(t *testing.T) {
	// Type 0x0c + length 0
	reader := hexReader("0c" + uint32Hex(0))
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Failed to decode empty tuple: %v", err)
	}
	tuple, ok := val.(*Tuple)
	if !ok {
		t.Fatalf("Expected *Tuple, got %T", val)
	}
	if tuple.Length != 0 {
		t.Errorf("Expected length 0, got %d", tuple.Length)
	}
	if len(tuple.Values) != 0 {
		t.Errorf("Expected empty values, got %d", len(tuple.Values))
	}
}

func TestMaxDepthExceeded(t *testing.T) {
	// Create 17 nested Some values: Some(Some(Some(...(true))))
	// MaxClarityDepth is 16, so this should fail
	// Each Some is: 0x0a (SomeOption type)
	// Innermost value is: 0x03 (true)

	// Build the hex string: 17 Some types + 1 true
	hexStr := ""
	for i := 0; i < 17; i++ {
		hexStr += "0a" // SomeOption
	}
	hexStr += "03" // BooleanTrue

	reader := hexReader(hexStr)
	_, err := DecodeClarityValue(reader)
	if err == nil {
		t.Fatal("Expected error for depth exceeding maximum, got nil")
	}
	expectedErr := "nesting depth"
	if !contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestMaxDepthAllowed(t *testing.T) {
	// Create exactly 16 nested Some values (the maximum allowed)
	// This should succeed

	// Build the hex string: 16 Some types + 1 true
	hexStr := ""
	for i := 0; i < 16; i++ {
		hexStr += "0a" // SomeOption
	}
	hexStr += "03" // BooleanTrue

	reader := hexReader(hexStr)
	val, err := DecodeClarityValue(reader)
	if err != nil {
		t.Fatalf("Expected success for depth at maximum, got error: %v", err)
	}

	// Verify we got an Option back
	if _, ok := val.(*Option); !ok {
		t.Errorf("Expected *Option, got %T", val)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
