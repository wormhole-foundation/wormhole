package stacks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
)

// Define an enum called ClarityType with the following values:
// 0x00: 128-bit signed integer
// 0x01: 128-bit unsigned integer
// 0x02: buffer
// 0x03: boolean true
// 0x04: boolean false
// 0x05: standard principal
// 0x06: contract principal
// 0x07: Ok response
// 0x08: Err response
// 0x09: None option
// 0x0a: Some option
// 0x0b: List
// 0x0c: Tuple
// 0x0d: StringASCII
// 0x0e: StringUTF8
type ClarityType uint8 //enums:enum

const (
	Int128Signed ClarityType = iota
	Int128Unsigned
	Buffer
	BooleanTrue
	BooleanFalse
	StandardPrincipal
	ContractPrincipal
	OkResponse
	ErrResponse
	NoneOption
	SomeOption
	ListType
	TupleType
	StringASCIIType
	StringUTF8Type
)

// Memory safety limits to prevent OOM denial of service attacks
const (
	MaxClarityBufferLength = 1024 * 1024 // 1MB - Maximum size for Buffer type
	MaxClarityStringLength = 1024 * 1024 // 1MB - Maximum size for ASCII/UTF8 strings
	MaxClarityListLength   = 10000       // Maximum number of elements in a List
	MaxClarityTupleLength  = 1000        // Maximum number of fields in a Tuple
	MaxClarityDepth        = 16          // Maximum nesting depth for Clarity values
)

type ClarityValue interface {
	ClarityDecode(*bytes.Reader) error
}

type Int128 struct {
	Value *big.Int
}

func (i *Int128) ClarityDecode(r *bytes.Reader) error {
	buf := make([]byte, 16)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}

	// SetBytes interprets as unsigned
	i.Value = new(big.Int).SetBytes(buf)

	// Check if high bit is set (two's complement negative)
	if buf[0]&0x80 != 0 {
		// Subtract 2^128 to get the proper signed value
		modulus := new(big.Int).Lsh(big.NewInt(1), 128)
		i.Value.Sub(i.Value, modulus)
	}

	return nil
}

type UInt128 struct {
	Value *big.Int
}

func (u *UInt128) ClarityDecode(r *bytes.Reader) error {
	buf := make([]byte, 16)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	u.Value = new(big.Int).SetBytes(buf)
	return nil
}

type ClarityBuffer struct {
	Length uint32
	Data   []byte
}

func (b *ClarityBuffer) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &b.Length)
	if err != nil {
		return err
	}
	if b.Length > MaxClarityBufferLength {
		return fmt.Errorf("clarity: buffer length %d exceeds maximum %d", b.Length, MaxClarityBufferLength)
	}
	b.Data = make([]byte, b.Length)
	_, err = io.ReadFull(r, b.Data)
	return err
}

type Bool struct {
	Value bool
}

func (b *Bool) ClarityDecode(r *bytes.Reader) error {
	var val uint8
	err := binary.Read(r, binary.BigEndian, &val)
	if err != nil {
		return err
	}
	b.Value = val != 0
	return nil
}

type ClarityPrincipal struct {
	Version byte
	Hash160 [20]byte
}

func (p *ClarityPrincipal) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &p.Version)
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, p.Hash160[:])
	return err
}

type ClarityContractPrincipal struct {
	ClarityPrincipal
	Name string
}

func (c *ClarityContractPrincipal) ClarityDecode(r *bytes.Reader) error {
	err := c.ClarityPrincipal.ClarityDecode(r)
	if err != nil {
		return err
	}
	var nameLength uint8
	err = binary.Read(r, binary.BigEndian, &nameLength)
	if err != nil {
		return err
	}
	nameBytes := make([]byte, nameLength)
	_, err = io.ReadFull(r, nameBytes)
	if err != nil {
		return err
	}
	c.Name = string(nameBytes)
	return nil
}

type Response struct {
	IsOk   bool
	Result ClarityValue
}

func (res *Response) clarityDecodeWithDepth(r *bytes.Reader, depth int) error {
	var err error
	res.Result, err = decodeClarityValueWithDepth(r, depth)
	return err
}

func (res *Response) ClarityDecode(r *bytes.Reader) error {
	return res.clarityDecodeWithDepth(r, 0)
}

type Option struct {
	IsSome bool
	Value  ClarityValue
}

func (opt *Option) clarityDecodeWithDepth(r *bytes.Reader, depth int) error {
	var err error
	opt.Value, err = decodeClarityValueWithDepth(r, depth)
	opt.IsSome = err == nil
	return err
}

func (opt *Option) ClarityDecode(r *bytes.Reader) error {
	return opt.clarityDecodeWithDepth(r, 0)
}

type List struct {
	Length uint32
	Values []ClarityValue
}

func (l *List) clarityDecodeWithDepth(r *bytes.Reader, depth int) error {
	err := binary.Read(r, binary.BigEndian, &l.Length)
	if err != nil {
		return err
	}
	if l.Length > MaxClarityListLength {
		return fmt.Errorf("clarity: list length %d exceeds maximum %d", l.Length, MaxClarityListLength)
	}
	l.Values = make([]ClarityValue, l.Length)
	for i := uint32(0); i < l.Length; i++ {

		// WARNING: Allows for different types within the List. Not valid Clarity though.
		l.Values[i], err = decodeClarityValueWithDepth(r, depth)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *List) ClarityDecode(r *bytes.Reader) error {
	return l.clarityDecodeWithDepth(r, 0)
}

type Tuple struct {
	Length uint32
	Values map[string]ClarityValue
}

func (t *Tuple) clarityDecodeWithDepth(r *bytes.Reader, depth int) error {
	err := binary.Read(r, binary.BigEndian, &t.Length)
	if err != nil {
		return err
	}
	if t.Length > MaxClarityTupleLength {
		return fmt.Errorf("clarity: tuple length %d exceeds maximum %d", t.Length, MaxClarityTupleLength)
	}
	t.Values = make(map[string]ClarityValue)
	for i := uint32(0); i < t.Length; i++ {
		var nameLength uint8
		err = binary.Read(r, binary.BigEndian, &nameLength)
		if err != nil {
			return err
		}
		nameBytes := make([]byte, nameLength)
		_, err = io.ReadFull(r, nameBytes)
		if err != nil {
			return err
		}
		name := string(nameBytes)
		t.Values[name], err = decodeClarityValueWithDepth(r, depth)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tuple) ClarityDecode(r *bytes.Reader) error {
	return t.clarityDecodeWithDepth(r, 0)
}

type StringASCII struct {
	Length uint32
	Value  string
}

func (s *StringASCII) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &s.Length)
	if err != nil {
		return err
	}
	if s.Length > MaxClarityStringLength {
		return fmt.Errorf("clarity: string-ascii length %d exceeds maximum %d", s.Length, MaxClarityStringLength)
	}

	// WARNING: No byte validation to ensure it's valid ASCII.
	valueBytes := make([]byte, s.Length)
	_, err = io.ReadFull(r, valueBytes)
	if err != nil {
		return err
	}
	s.Value = string(valueBytes)
	return nil
}

type StringUTF8 struct {
	Length uint32
	Value  string
}

func (s *StringUTF8) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &s.Length)
	if err != nil {
		return err
	}
	if s.Length > MaxClarityStringLength {
		return fmt.Errorf("clarity: string-utf8 length %d exceeds maximum %d", s.Length, MaxClarityStringLength)
	}
	valueBytes := make([]byte, s.Length)
	_, err = io.ReadFull(r, valueBytes)
	if err != nil {
		return err
	}
	s.Value = string(valueBytes)
	return nil
}

func decodeClarityValueWithDepth(r *bytes.Reader, depth int) (ClarityValue, error) {
	if depth > MaxClarityDepth {
		return nil, fmt.Errorf("clarity: nesting depth %d exceeds maximum %d", depth, MaxClarityDepth)
	}

	var typeID uint8
	err := binary.Read(r, binary.BigEndian, &typeID)
	if err != nil {
		return nil, err
	}

	switch ClarityType(typeID) {
	case Int128Signed:
		val := &Int128{}
		err := val.ClarityDecode(r)
		return val, err
	case Int128Unsigned:
		val := &UInt128{}
		err := val.ClarityDecode(r)
		return val, err
	case Buffer:
		val := &ClarityBuffer{}
		err := val.ClarityDecode(r)
		return val, err
	case BooleanTrue, BooleanFalse:
		val := &Bool{}
		val.Value = typeID == uint8(BooleanTrue)
		return val, nil
	case StandardPrincipal:
		val := &ClarityPrincipal{}
		err := val.ClarityDecode(r)
		return val, err
	case ContractPrincipal:
		val := &ClarityContractPrincipal{}
		err := val.ClarityDecode(r)
		return val, err
	case OkResponse, ErrResponse:
		val := &Response{}
		err := val.clarityDecodeWithDepth(r, depth+1)
		val.IsOk = typeID == uint8(OkResponse)
		return val, err
	case NoneOption:
		// None has no value, don't read any additional data
		return &Option{IsSome: false, Value: nil}, nil
	case SomeOption:
		// Some contains a value, decode it
		val := &Option{IsSome: true}
		val.Value, err = decodeClarityValueWithDepth(r, depth+1)
		return val, err
	case ListType:
		val := &List{}
		err := val.clarityDecodeWithDepth(r, depth+1)
		return val, err
	case TupleType:
		val := &Tuple{}
		err := val.clarityDecodeWithDepth(r, depth+1)
		return val, err
	case StringASCIIType:
		val := &StringASCII{}
		err := val.ClarityDecode(r)
		return val, err
	case StringUTF8Type:
		val := &StringUTF8{}
		err := val.ClarityDecode(r)
		return val, err
	default:
		return nil, fmt.Errorf("unknown clarity type ID: 0x%02x", typeID)
	}
}

// DecodeClarityValue decodes a Clarity value from a bytes.Reader.
// This is the public entry point that enforces recursion depth limits.
func DecodeClarityValue(r *bytes.Reader) (ClarityValue, error) {
	return decodeClarityValueWithDepth(r, 0)
}
