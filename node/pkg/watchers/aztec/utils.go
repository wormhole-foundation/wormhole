package aztec

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Helper functions for common parsing tasks

// ParseUint parses a string as an unsigned integer with proper error handling
func ParseUint(s string, base int, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		return 0, &ErrParsingFailed{
			What: fmt.Sprintf("unsigned integer with base %d", base),
			Err:  err,
		}
	}
	return v, nil
}

func (e *ErrParsingFailed) Unwrap() error {
	return e.Err
}

// GetJSONRPCError extracts error information from a JSON-RPC response
func GetJSONRPCError(body []byte) (bool, *ErrRPCError) {
	var errorCheck struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &errorCheck); err != nil || errorCheck.Error == nil {
		return false, nil
	}

	return true, &ErrRPCError{
		Method: "unknown", // Caller should update this
		Code:   errorCheck.Error.Code,
		Msg:    errorCheck.Error.Message,
	}
}

// CreateObservationID creates a unique ID for tracking pending observations
func CreateObservationID(senderAddress string, sequence uint64, blockNumber int) string {
	return fmt.Sprintf("%s-%d-%d", senderAddress, sequence, blockNumber)
}
