package msg

import sdk "github.com/cosmos/cosmos-sdk/types"

// Types - wrappers to expose interface
type (
	Coin     = sdk.Coin
	Coins    = sdk.Coins
	DecCoin  = sdk.DecCoin
	DecCoins = sdk.DecCoins

	Int = sdk.Int
	Dec = sdk.Dec

	AccAddress  = sdk.AccAddress
	ValAddress  = sdk.ValAddress
	ConsAddress = sdk.ConsAddress
)

// AccAddressFromBech32 creates an AccAddress from a Bech32 string.
func AccAddressFromBech32(address string) (addr AccAddress, err error) {
	return sdk.AccAddressFromBech32(address)
}

// ValAddressFromBech32 creates a ValAddress from a Bech32 string.
func ValAddressFromBech32(address string) (addr ValAddress, err error) {
	return sdk.ValAddressFromBech32(address)
}

// ConsAddressFromBech32 creates a ConsAddress from a Bech32 string.
func ConsAddressFromBech32(address string) (addr ConsAddress, err error) {
	return sdk.ConsAddressFromBech32(address)
}

// NewCoin returns a new coin with a denomination and amount. It will panic if
// the amount is negative.
func NewCoin(denom string, amount Int) Coin {
	return sdk.NewCoin(denom, amount)
}

// NewInt64Coin returns a new coin with a denomination and amount. It will panic
// if the amount is negative.
func NewInt64Coin(denom string, amount int64) Coin {
	return sdk.NewInt64Coin(denom, amount)
}

// NewCoins constructs a new coin set.
func NewCoins(coins ...Coin) Coins {
	return sdk.NewCoins(coins...)
}

// NewInt constructs Int from int64
func NewInt(n int64) Int {
	return sdk.NewInt(n)
}

// NewIntFromUint64 constructs an Int from a uint64.
func NewIntFromUint64(n uint64) Int {
	return sdk.NewIntFromUint64(n)
}

// NewIntFromString constructs Int from string
func NewIntFromString(s string) (res Int, ok bool) {
	return sdk.NewIntFromString(s)
}

// NewIntWithDecimal constructs Int with decimal
// Result value is n*10^dec
func NewIntWithDecimal(n int64, dec int) Int {
	return sdk.NewIntWithDecimal(n, dec)
}

// NewDec create a new Dec from integer assuming whole number
func NewDec(i int64) Dec {
	return sdk.NewDec(i)
}

// NewDecWithPrec create a new Dec from integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecWithPrec(i, prec int64) Dec {
	return sdk.NewDecWithPrec(i, prec)
}

// NewDecFromInt create a new Dec from big integer assuming whole numbers
// CONTRACT: prec <= Precision
func NewDecFromInt(i Int) Dec {
	return sdk.NewDecFromInt(i)
}

// NewDecFromIntWithPrec create a new Dec from big integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecFromIntWithPrec(i Int, prec int64) Dec {
	return sdk.NewDecFromIntWithPrec(i, prec)
}

// NewDecFromStr create a decimal from an input decimal string.
// valid must come in the form:
//   (-) whole integers (.) decimal integers
// examples of acceptable input include:
//   -123.456
//   456.7890
//   345
//   -456789
//
// NOTE - An error will return if more decimal places
// are provided in the string than the constant Precision.
//
// CONTRACT - This function does not mutate the input str.
func NewDecFromStr(str string) (Dec, error) {
	return sdk.NewDecFromStr(str)
}

// NewDecCoin creates a new DecCoin instance from an Int.
func NewDecCoin(denom string, amount Int) DecCoin {
	return sdk.NewDecCoin(denom, amount)
}

// NewDecCoinFromDec creates a new DecCoin instance from a Dec.
func NewDecCoinFromDec(denom string, amount Dec) DecCoin {
	return sdk.NewDecCoinFromDec(denom, amount)
}

// NewDecCoinFromCoin creates a new DecCoin from a Coin.
func NewDecCoinFromCoin(coin Coin) DecCoin {
	return sdk.NewDecCoinFromCoin(coin)
}

// NewInt64DecCoin returns a new DecCoin with a denomination and amount. It will
// panic if the amount is negative or denom is invalid.
func NewInt64DecCoin(denom string, amount int64) DecCoin {
	return NewInt64DecCoin(denom, amount)
}
