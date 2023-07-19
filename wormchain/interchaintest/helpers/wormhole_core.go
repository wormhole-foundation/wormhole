package helpers

import (
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type CoreInstantiateMsg struct {
	GovChain            uint16          `json:"gov_chain"`
	GovAddress          []byte          `json:"gov_address"`
	InitialGuardianSet  GuardianSetInfo `json:"initial_guardian_set"`
	GuardianSetExpirity uint64          `json:"guardian_set_expirity"`
	ChainId             uint16          `json:"chain_id"`
	FeeDenom            string          `json:"fee_denom"`
}

type GuardianSetInfo struct {
	Addresses      []GuardianAddress `json:"addresses"`
	ExpirationTime uint64            `json:"expiration_time"`
}

type GuardianAddress struct {
	Bytes []byte `json:"bytes"`
}

func CoreContractInstantiateMsg(t *testing.T, cfg ibc.ChainConfig, guardians *guardians.ValSet) string {
	guardianAddresses := []GuardianAddress{}
	for i := 0; i < guardians.Total; i++ {
		guardianAddresses = append(guardianAddresses, GuardianAddress{
			Bytes: guardians.Vals[i].Addr,
		})
	}

	msg := CoreInstantiateMsg{
		GovChain:   uint16(vaa.GovernanceChain),
		GovAddress: vaa.GovernanceEmitter[:],
		InitialGuardianSet: GuardianSetInfo{
			Addresses:      guardianAddresses,
			ExpirationTime: 0,
		},
		GuardianSetExpirity: 86400,
		ChainId:             uint16(vaa.ChainIDWormchain),
		FeeDenom:            cfg.Denom,
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}
