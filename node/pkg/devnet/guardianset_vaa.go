package devnet

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/ethereum/abi"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

// DevnetGuardianSetVSS returns a VAA signed by guardian-0 that adds all n validators.
func DevnetGuardianSetVSS(n uint) *vaa.VAA {
	pubkeys := make([]common.Address, n)

	for n := range pubkeys {
		key := InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), uint64(n))
		pubkeys[n] = crypto.PubkeyToAddress(key.PublicKey)
	}

	v := &vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(5000, 0),
		EmitterChain:     3,
		EmitterAddress: vaa.Address{
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
		},
		Payload: vaa.BodyGuardianSetUpdate{
			Keys:     pubkeys,
			NewIndex: 1,
		}.Serialize(),
	}

	// The devnet is initialized with a single guardian (ethereum/migrations/1_initial_migration.js).
	key0 := InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), 0)
	v.AddSignature(key0, 0)

	return v
}

// SubmitVAA submits a VAA to the devnet chain using well-known accounts and contract addresses.
func SubmitVAA(ctx context.Context, rpcURL string, vaa *vaa.VAA) (*types.Transaction, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dialing eth client failed: %w", err)
	}

	kt := GetKeyedTransactor(ctx)
	contract, err := abi.NewAbi(GanacheWormholeContractAddress, c)
	if err != nil {
		panic(err)
	}

	b, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}

	supervisor.Logger(ctx).Info("submitted VAA to Ethereum devnet", zap.Binary("binary", b))

	tx, err := contract.SubmitNewGuardianSet(kt, b)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// GetKeyedTransactor returns a transaction signer with the deterministic devnet key.
func GetKeyedTransactor(ctx context.Context) *bind.TransactOpts {
	key, err := Wallet().PrivateKey(DeriveAccount(0))
	if err != nil {
		panic(err)
	}

	kt := bind.NewKeyedTransactor(key) // nolint
	kt.Context = ctx

	return kt
}
