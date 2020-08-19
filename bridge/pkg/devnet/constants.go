// package devnet contains constants and helper functions for the local deterministic devnet.
package devnet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"

	"github.com/miguelmota/go-ethereum-hdwallet"
)

var (
	// Address of the first account, which is used as the default client account.
	GanacheClientDefaultAccountAddress = common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")

	// Contracts (deployed by "truffle migrate" on a deterministic devnet)
	WrappedAssetContractAddress = common.HexToAddress("0x79183957Be84C0F4dA451E534d5bA5BA3FB9c696")
	BridgeContractAddress       = common.HexToAddress("0xCfEB869F69431e42cdB54A4F4f105C19C080A601")
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
