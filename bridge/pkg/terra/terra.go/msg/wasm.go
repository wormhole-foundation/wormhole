package msg

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ Msg = StoreCode{}
var _ Msg = InstantiateContract{}
var _ Msg = ExecuteContract{}

// StoreCodeData - value part of StoreCode
type StoreCodeData struct {
	Sender       AccAddress `json:"sender"`
	WASMByteCode []byte     `json:"wasm_byte_code"`
}

// StoreCode - high level transaction of the wasm module
type StoreCode struct {
	Type  string        `json:"type"`
	Value StoreCodeData `json:"value"`
}

// NewStoreCode - create StoreCode
func NewStoreCode(sender AccAddress, wasmByteCode []byte) StoreCode {
	return StoreCode{
		Type: "wasm/MsgStoreCode",
		Value: StoreCodeData{
			Sender:       sender,
			WASMByteCode: wasmByteCode,
		},
	}
}

// GetType - Msg interface
func (m StoreCode) GetType() string {
	return "store_code"
}

// GetSignBytes - Msg interface
func (m StoreCode) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m StoreCode) GetSigners() []AccAddress {
	return []AccAddress{m.Value.Sender}
}

// GetSendCoins - return send coins for tax calulcation
func (m StoreCode) GetSendCoins() Coins {
	return Coins{}
}

// InstantiateContractData - value part of InstantiateContract
type InstantiateContractData struct {
	Owner      AccAddress `json:"owner"`
	CodeID     uint64     `json:"code_id"`
	InitMsg    []byte     `json:"init_msg"`
	InitCoins  sdk.Coins  `json:"init_coins"`
	Migratable bool       `json:",migratable"`
}

// InstantiateContract - high level transaction of the wasm module
type InstantiateContract struct {
	Type  string                  `json:"type"`
	Value InstantiateContractData `json:"value"`
}

// NewInstantiateContract - create InstantiateContract
func NewInstantiateContract(owner AccAddress, codeID uint64, initMsg []byte, initCoins sdk.Coins, migratable bool) InstantiateContract {
	return InstantiateContract{
		Type: "wasm/MsgInstantiateContract",
		Value: InstantiateContractData{
			Owner:      owner,
			CodeID:     codeID,
			InitMsg:    initMsg,
			InitCoins:  initCoins,
			Migratable: migratable,
		},
	}
}

// GetType - Msg interface
func (m InstantiateContract) GetType() string {
	return "instantiate_contract"
}

// GetSignBytes - Msg interface
func (m InstantiateContract) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m InstantiateContract) GetSigners() []AccAddress {
	return []AccAddress{m.Value.Owner}
}

// GetSendCoins - return send coins for tax calulcation
func (m InstantiateContract) GetSendCoins() Coins {
	return m.Value.InitCoins
}

// ExecuteContractData - value part of ExecuteContract
type ExecuteContractData struct {
	Sender     AccAddress `json:"sender"`
	Contract   AccAddress `json:"contract"`
	ExecuteMsg []byte     `json:"execute_msg"`
	Coins      sdk.Coins  `json:"coins"`
}

// ExecuteContract - high level transaction of the wasm module
type ExecuteContract struct {
	Type  string              `json:"type"`
	Value ExecuteContractData `json:"value"`
}

// NewExecuteContract - create ExecuteContract
func NewExecuteContract(sender, contract AccAddress, executeMsg []byte, coins sdk.Coins) ExecuteContract {
	return ExecuteContract{
		Type: "wasm/MsgExecuteContract",
		Value: ExecuteContractData{
			Sender:     sender,
			Contract:   contract,
			ExecuteMsg: executeMsg,
			Coins:      coins,
		},
	}
}

// GetType - Msg interface
func (m ExecuteContract) GetType() string {
	return "execute_contract"
}

// GetSignBytes - Msg interface
func (m ExecuteContract) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m ExecuteContract) GetSigners() []AccAddress {
	return []AccAddress{m.Value.Sender}
}

// GetSendCoins - return send coins for tax calulcation
func (m ExecuteContract) GetSendCoins() Coins {
	return m.Value.Coins
}
