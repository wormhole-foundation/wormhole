package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Config:                         nil,
		ReplayProtectionList:           []ReplayProtection{},
		ChainRegistrationList:          []ChainRegistration{},
		CoinMetaRollbackProtectionList: []CoinMetaRollbackProtection{},
		// this line is used by starport scaffolding # genesis/types/default
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in replayProtection
	replayProtectionIndexMap := make(map[string]struct{})

	for _, elem := range gs.ReplayProtectionList {
		index := string(ReplayProtectionKey(elem.Index))
		if _, ok := replayProtectionIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for replayProtection")
		}
		replayProtectionIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in chainRegistration
	chainRegistrationIndexMap := make(map[string]struct{})

	for _, elem := range gs.ChainRegistrationList {
		index := string(ChainRegistrationKey(elem.ChainID))
		if _, ok := chainRegistrationIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for chainRegistration")
		}
		chainRegistrationIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in coinMetaRollbackProtection
	coinMetaRollbackProtectionIndexMap := make(map[string]struct{})

	for _, elem := range gs.CoinMetaRollbackProtectionList {
		index := string(CoinMetaRollbackProtectionKey(elem.Index))
		if _, ok := coinMetaRollbackProtectionIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for coinMetaRollbackProtection")
		}
		coinMetaRollbackProtectionIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return nil
}
