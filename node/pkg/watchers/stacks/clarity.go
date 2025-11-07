package stacks

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	i.Value = new(big.Int).SetBytes(buf)
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
	b.Data = make([]byte, b.Length)
	_, err = r.Read(b.Data)
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

func (res *Response) ClarityDecode(r *bytes.Reader) error {
	var err error
	res.Result, err = DecodeClarityValue(r)
	res.IsOk = err == nil
	return err
}

type Option struct {
	IsSome bool
	Value  ClarityValue
}

func (opt *Option) ClarityDecode(r *bytes.Reader) error {
	var err error
	opt.Value, err = DecodeClarityValue(r)
	opt.IsSome = err == nil
	return err
}

type List struct {
	Length uint32
	Values []ClarityValue
}

func (l *List) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &l.Length)
	if err != nil {
		return err
	}
	l.Values = make([]ClarityValue, l.Length)
	for i := uint32(0); i < l.Length; i++ {
		l.Values[i], err = DecodeClarityValue(r)
		if err != nil {
			return err
		}
	}
	return nil
}

type Tuple struct {
	Length uint32
	Values map[string]ClarityValue
}

func (t *Tuple) ClarityDecode(r *bytes.Reader) error {
	err := binary.Read(r, binary.BigEndian, &t.Length)
	if err != nil {
		return err
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
		t.Values[name], err = DecodeClarityValue(r)
		if err != nil {
			return err
		}
	}
	return nil
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
	valueBytes := make([]byte, s.Length)
	_, err = io.ReadFull(r, valueBytes)
	if err != nil {
		return err
	}
	s.Value = string(valueBytes)
	return nil
}

func DecodeClarityValue(r *bytes.Reader) (ClarityValue, error) {
	var typeID uint8
	err := binary.Read(r, binary.BigEndian, &typeID)
	if err != nil {
		return nil, err
	}

	switch typeID {
	case 0x00:
		val := &Int128{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x01:
		val := &UInt128{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x02:
		val := &ClarityBuffer{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x03, 0x04:
		val := &Bool{}
		// r.Seek(-1, 1) // patch: Seek back 1 byte from current position if we need to decode the value
		// err := val.ClarityDecode(r)
		val.Value = typeID == 0x03
		// return val, err
		return val, nil
	case 0x05:
		val := &ClarityPrincipal{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x06:
		val := &ClarityContractPrincipal{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x07, 0x08:
		val := &Response{}
		err := val.ClarityDecode(r)
		val.IsOk = typeID == 0x07
		return val, err
	case 0x09, 0x0a:
		val := &Option{}
		err := val.ClarityDecode(r)
		val.IsSome = typeID == 0x0a
		return val, err
	case 0x0b:
		val := &List{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x0c:
		val := &Tuple{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x0d:
		val := &StringASCII{}
		err := val.ClarityDecode(r)
		return val, err
	case 0x0e:
		val := &StringUTF8{}
		err := val.ClarityDecode(r)
		return val, err
	default:
		return nil, errors.New("unknown type ID")
	}
}
