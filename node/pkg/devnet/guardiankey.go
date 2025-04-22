package devnet

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// GenerateAndStoreDevnetGuardianKey returns a deterministic testnet key.
func GenerateAndStoreDevnetGuardianKey(filename string) error {
	// Figure out our devnet index
	idx, err := GetDevnetIndex()
	if err != nil {
		return err
	}

	// Generate the guardian key.
	gk := InsecureDeterministicEcdsaKeyByIndex(ethcrypto.S256(), uint64(idx)) // #nosec G115 -- Number of guardians will never overflow here

	// Store it to disk.
	if err := common.WriteArmoredKey(gk, "auto-generated deterministic devnet key", filename, common.GuardianKeyArmoredBlock, true); err != nil {
		return fmt.Errorf("failed to store generated guardian key: %w", err)
	}

	return nil
}
