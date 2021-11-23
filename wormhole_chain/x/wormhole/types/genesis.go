package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		GuardianSetList:      []GuardianSet{},
		Config:               nil,
		ReplayProtectionList: []ReplayProtection{},
		// this line is used by starport scaffolding # genesis/types/default
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated ID in guardianSet
	guardianSetIdMap := make(map[uint32]bool)
	guardianSetCount := gs.GetGuardianSetCount()
	for _, elem := range gs.GuardianSetList {
		if _, ok := guardianSetIdMap[elem.Index]; ok {
			return fmt.Errorf("duplicated id for guardianSet")
		}
		if elem.Index >= guardianSetCount {
			return fmt.Errorf("guardianSet id should be lower or equal than the last id")
		}
		guardianSetIdMap[elem.Index] = true
	}
	// Check for duplicated index in replayProtection
	replayProtectionIndexMap := make(map[string]struct{})

	for _, elem := range gs.ReplayProtectionList {
		index := string(ReplayProtectionKey(elem.Index))
		if _, ok := replayProtectionIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for replayProtection")
		}
		replayProtectionIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return nil
}
