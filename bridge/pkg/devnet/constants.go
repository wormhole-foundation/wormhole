// package devnet contains constants and helper functions for the local deterministic devnet.
// See "Devnet addresses" in DEVELOP.md. Created by setup scripts/migrations.
package devnet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

var (
	// Ganache RPC URL
	GanacheRPCURL = "ws://localhost:8545"

	// Address of the first account, which is used as the default client account.
	GanacheClientDefaultAccountAddress = common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")

	// Contracts (deployed by "truffle migrate" on a deterministic devnet)
	GanacheAssetContractAddress  = common.HexToAddress("0xe78A0F7E598Cc8b0Bb87894B0F60dD2a88d6a8Ab")
	GanacheBridgeContractAddress = common.HexToAddress("0x5b1869D9A4C187F2EAa108f3062412ecf0526b24")

	// ERC20 example tokens.
	GanacheExampleERC20Token      = common.HexToAddress("0xCfEB869F69431e42cdB54A4F4f105C19C080A601")
	GanacheExampleERC20WrappedSOL = common.HexToAddress("0xf5b1d8fab1054b9cf7db274126972f97f9d42a11")
)

const (
	// Ganache's hardcoded HD Wallet derivation path
	ganacheWalletMnemonic = "myth like bonus scare over problem client lizard pioneer submit female collect"
	ganacheDerivationPath = "m/44'/60'/0'/0/%d"
)

const (
	// id.json account filled by faucet.
	SolanaCLIAccount = "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J"

	// Hardcoded contract addresses.
	SolanaBridgeContract = "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
	SolanaTokenContract  = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"

	// Native SPL token
	SolanaExampleToken              = "6qRhs8oAuZYLd4zzaNnQHqdRyknrQQWDWQhALEN8UA7M"
	SolanaExampleTokenOwningAccount = "3C3m4tjTy4nSMkkYdqCDSiCWEgpDa6whvprvABdFGBiW"

	// Wrapped ERC20 token
	SolanaExampleWrappedERCToken              = "85kW19uNvETzH43p3AfpyqPaQS5rWouq4x9rGiKUvihf"
	SolanaExampleWrappedERCTokenOwningAccount = "7EFk3VrWeb29SWJPQs5cUyqcY3fQd33S9gELkGybRzeu"

	// Lamports per SOL.
	SolanaDefaultPrecision = 1e9

	// ERC20 default precision.
	ERC20DefaultPrecision = 1e18
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
