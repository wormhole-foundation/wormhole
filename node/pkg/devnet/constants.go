// package devnet contains constants and helper functions for the local deterministic devnet.
// See "Devnet addresses" in DEVELOP.md. Created by setup scripts/migrations.
package devnet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

var (
	// Ganache RPC URL
	GanacheRPCURL = "ws://localhost:8545"

	// Address of the first account, which is used as the default client account.
	GanacheClientDefaultAccountAddress = common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")

	// Contracts (deployed by "truffle migrate" on a deterministic devnet)
	GanacheWormholeContractAddress = common.HexToAddress("0xC89Ce4735882C9F0f0FE26686c53074E09B0D550")
)

const (
	// Ganache's hardcoded HD Wallet derivation path
	ganacheWalletMnemonic = "myth like bonus scare over problem client lizard pioneer submit female collect"
	ganacheDerivationPath = "m/44'/60'/0'/0/%d"
)

func DeriveAccount(accountIndex uint) accounts.Account {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf(ganacheDerivationPath, accountIndex))
	account, err := Wallet().Derive(path, false)
	if err != nil {
		panic(err)
	}

	return account
}

func Wallet() *hdwallet.Wallet {
	wallet, err := hdwallet.NewFromMnemonic(ganacheWalletMnemonic)
	if err != nil {
		panic(err)
	}
	return wallet
}

// Base58ToEthAddress converts a Solana base58 address to a 32-byte vaa.Address.
// Panics if the input data is invalid - intended for use on constants.
func MustBase58ToEthAddress(address string) (res vaa.Address) {
	b, err := base58.Decode(address)
	if err != nil {
		panic(err)
	}

	if n := copy(res[:], b); n != 32 {
		panic("invalid length")
	}
	return
}
