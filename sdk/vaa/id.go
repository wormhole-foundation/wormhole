package vaa

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrEmptyVAAID         = errors.New("VAA ID is empty")
	ErrInvalidVAAIDFormat = errors.New("VAA ID must be in the format chainId/emitterAddress/sequence")
)

// VAAID is the canonical Wormhole message identifier.
// It is the tuple emitter_chain/emitter_address/sequence.
type VAAID struct {
	EmitterChain   ChainID
	EmitterAddress Address
	Sequence       uint64
}

// String returns the canonical chain/address/sequence representation.
func (id VAAID) String() string {
	return fmt.Sprintf("%d/%s/%d", uint16(id.EmitterChain), id.EmitterAddress, id.Sequence)
}

// Empty reports whether every field is its zero value.
func (id VAAID) Empty() bool {
	return id == (VAAID{})
}

// Validate returns an error if the ID is not usable as a full Wormhole message identifier.
func (id VAAID) Validate() error {
	if id.EmitterChain == ChainIDUnset {
		return errors.New("VAA ID emitter chain is unset")
	}
	if id.EmitterAddress == (Address{}) {
		return errors.New("VAA ID emitter address is zero")
	}
	return nil
}

// VAAIDFromVAA returns the canonical VAA ID for the supplied VAA.
func VAAIDFromVAA(v *VAA) VAAID {
	if v == nil {
		return VAAID{}
	}
	return v.ID()
}

// ID returns the canonical VAA ID for the supplied VAA.
func (v *VAA) ID() VAAID {
	return VAAID{
		EmitterChain:   v.EmitterChain,
		EmitterAddress: v.EmitterAddress,
		Sequence:       v.Sequence,
	}
}

// NewVAAID constructs a VAA ID from validated numeric chain and string address fields.
// This method does not validate that the ChainID is known to the SDK. It only converts the emitterChain parameter
// to the correct type.
func NewVAAID[N number](emitterChain N, emitterAddress string, sequence uint64) (VAAID, error) {
	chainID, err := ChainIDFromNumber(emitterChain)
	if err != nil {
		return VAAID{}, err
	}

	address, err := parseVAAIDAddress(emitterAddress)
	if err != nil {
		return VAAID{}, err
	}

	id := VAAID{
		EmitterChain:   chainID,
		EmitterAddress: address,
		Sequence:       sequence,
	}

	if err := id.Validate(); err != nil {
		return VAAID{}, err
	}

	return id, nil
}

// VAAIDFromString parses a chain/address/sequence identifier into a validated VAA ID.
// Numeric chain IDs do not need to correspond to a chain registered in the SDK.
func VAAIDFromString(s string) (VAAID, error) {
	return parseVAAIDString(s, false)
}

// VAAIDFromStringKnownChain parses a chain/address/sequence identifier into a validated VAA ID.
// The chain component must correspond to a chain registered in the SDK.
func VAAIDFromStringKnownChain(s string) (VAAID, error) {
	return parseVAAIDString(s, true)
}

func parseVAAIDString(s string, knownChain bool) (VAAID, error) {
	if s == "" {
		return VAAID{}, ErrEmptyVAAID
	}

	parts := strings.Split(s, "/")
	if len(parts) != VAAIDPartsLen {
		return VAAID{}, ErrInvalidVAAIDFormat
	}

	chainID, err := parseVAAIDChain(parts[0], knownChain)
	if err != nil {
		return VAAID{}, err
	}

	address, err := parseVAAIDAddress(parts[1])
	if err != nil {
		return VAAID{}, err
	}

	sequence, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return VAAID{}, err
	}

	id := VAAID{
		EmitterChain:   chainID,
		EmitterAddress: address,
		Sequence:       sequence,
	}

	if err := id.Validate(); err != nil {
		return VAAID{}, err
	}

	return id, nil
}

func parseVAAIDChain(s string, knownChain bool) (ChainID, error) {
	if s == "" {
		return ChainIDUnset, errors.New("VAA ID emitter chain is empty")
	}
	if knownChain {
		return StringToKnownChainID(s)
	}

	return StringToChainID(s)
}

func parseVAAIDAddress(s string) (Address, error) {
	trimmed := strings.TrimPrefix(s, "0x")
	if len(trimmed) != AddressHexLen {
		return Address{}, errors.New("VAA ID emitter address must be 32 bytes")
	}

	return StringToAddress(trimmed)
}
