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
	GanacheBridgeContractAddress = common.HexToAddress("0x254dffcd3277c0b1660f6d42efbb754edababc2b")

	// ERC20 example tokens.
	GanacheExampleERC20Token        = common.HexToAddress("0xCfEB869F69431e42cdB54A4F4f105C19C080A601")
	GanacheExampleERC20WrappedSOL   = common.HexToAddress("0xf5b1d8fab1054b9cf7db274126972f97f9d42a11")
	GanacheExampleERC20WrappedTerra = common.HexToAddress("0x62b47a23cd900da982bdbe75aeb891d3ed18cc36")
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

	// Wrapped CW20 token
	SolanaExampleWrappedCWToken              = "9ESkHLgJH4zqbG7fvhpC9u2ZeHMoLJznCHtaRLviEVRh"
	SolanaExampleWrappedCWTokenOwningAccount = "EERzaqe8Agm8p1ZkGQFq9zKpP7MDW29FX1pC1vEw9Yfv"

	// Lamports per SOL.
	SolanaDefaultPrecision = 1e9

	// ERC20 default precision.
	ERC20DefaultPrecision = 1e18

	// CW20 default precision.
	TerraDefaultPrecision = 1e8

	// Terra LCD url
	TerraLCDURL = "http://localhost:1317"

	// Terra test chain ID
	TerraChainID = "localterra"

	// Terra main test address to send/receive tokens
	TerraMainTestAddress    = "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"
	TerraMainTestAddressHex = "00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d"

	// Terra token address
	TerraTokenAddress = "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5"

	// Terra bridge contract address
	TerraBridgeAddress = "terra174kgn5rtw4kf6f938wm7kwh70h2v4vcfd26jlc"

	// Terra devnet fee payer mnemonic
	TerraFeePayerKey = "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius"
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
