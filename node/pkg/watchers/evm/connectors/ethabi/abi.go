// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ethabi

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// GovernanceStructsContractUpgrade is an auto generated low-level Go binding around an user-defined struct.
type GovernanceStructsContractUpgrade struct {
	Module      [32]byte
	Action      uint8
	Chain       uint16
	NewContract common.Address
}

// GovernanceStructsGuardianSetUpgrade is an auto generated low-level Go binding around an user-defined struct.
type GovernanceStructsGuardianSetUpgrade struct {
	Module              [32]byte
	Action              uint8
	Chain               uint16
	NewGuardianSet      StructsGuardianSet
	NewGuardianSetIndex uint32
}

// GovernanceStructsSetMessageFee is an auto generated low-level Go binding around an user-defined struct.
type GovernanceStructsSetMessageFee struct {
	Module     [32]byte
	Action     uint8
	Chain      uint16
	MessageFee *big.Int
}

// GovernanceStructsTransferFees is an auto generated low-level Go binding around an user-defined struct.
type GovernanceStructsTransferFees struct {
	Module    [32]byte
	Action    uint8
	Chain     uint16
	Amount    *big.Int
	Recipient [32]byte
}

// StructsGuardianSet is an auto generated low-level Go binding around an user-defined struct.
type StructsGuardianSet struct {
	Keys           []common.Address
	ExpirationTime uint32
}

// StructsSignature is an auto generated low-level Go binding around an user-defined struct.
type StructsSignature struct {
	R             [32]byte
	S             [32]byte
	V             uint8
	GuardianIndex uint8
}

// StructsVM is an auto generated low-level Go binding around an user-defined struct.
type StructsVM struct {
	Version          uint8
	Timestamp        uint32
	Nonce            uint32
	EmitterChainId   uint16
	EmitterAddress   [32]byte
	Sequence         uint64
	ConsistencyLevel uint8
	Payload          []byte
	GuardianSetIndex uint32
	Signatures       []StructsSignature
	Hash             [32]byte
}

// AbiABI is the input ABI used to generate the binding from.
const AbiABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"beacon\",\"type\":\"address\"}],\"name\":\"BeaconUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldContract\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newContract\",\"type\":\"address\"}],\"name\":\"ContractUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"index\",\"type\":\"uint32\"}],\"name\":\"GuardianSetAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"sequence\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"consistencyLevel\",\"type\":\"uint8\"}],\"name\":\"LogMessagePublished\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"chainId\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentGuardianSetIndex\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"index\",\"type\":\"uint32\"}],\"name\":\"getGuardianSet\",\"outputs\":[{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expirationTime\",\"type\":\"uint32\"}],\"internalType\":\"structStructs.GuardianSet\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGuardianSetExpiry\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"governanceActionIsConsumed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"governanceChainId\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"governanceContract\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"impl\",\"type\":\"address\"}],\"name\":\"isInitialized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"emitter\",\"type\":\"address\"}],\"name\":\"nextSequence\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedVM\",\"type\":\"bytes\"}],\"name\":\"parseAndVerifyVM\",\"outputs\":[{\"components\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"internalType\":\"uint16\",\"name\":\"emitterChainId\",\"type\":\"uint16\"},{\"internalType\":\"bytes32\",\"name\":\"emitterAddress\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"sequence\",\"type\":\"uint64\"},{\"internalType\":\"uint8\",\"name\":\"consistencyLevel\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"guardianSetIndex\",\"type\":\"uint32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"guardianIndex\",\"type\":\"uint8\"}],\"internalType\":\"structStructs.Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"}],\"internalType\":\"structStructs.VM\",\"name\":\"vm\",\"type\":\"tuple\"},{\"internalType\":\"bool\",\"name\":\"valid\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedUpgrade\",\"type\":\"bytes\"}],\"name\":\"parseContractUpgrade\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"module\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"action\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"chain\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"newContract\",\"type\":\"address\"}],\"internalType\":\"structGovernanceStructs.ContractUpgrade\",\"name\":\"cu\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedUpgrade\",\"type\":\"bytes\"}],\"name\":\"parseGuardianSetUpgrade\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"module\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"action\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"chain\",\"type\":\"uint16\"},{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expirationTime\",\"type\":\"uint32\"}],\"internalType\":\"structStructs.GuardianSet\",\"name\":\"newGuardianSet\",\"type\":\"tuple\"},{\"internalType\":\"uint32\",\"name\":\"newGuardianSetIndex\",\"type\":\"uint32\"}],\"internalType\":\"structGovernanceStructs.GuardianSetUpgrade\",\"name\":\"gsu\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedSetMessageFee\",\"type\":\"bytes\"}],\"name\":\"parseSetMessageFee\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"module\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"action\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"chain\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"messageFee\",\"type\":\"uint256\"}],\"internalType\":\"structGovernanceStructs.SetMessageFee\",\"name\":\"smf\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedTransferFees\",\"type\":\"bytes\"}],\"name\":\"parseTransferFees\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"module\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"action\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"chain\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"}],\"internalType\":\"structGovernanceStructs.TransferFees\",\"name\":\"tf\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"encodedVM\",\"type\":\"bytes\"}],\"name\":\"parseVM\",\"outputs\":[{\"components\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"internalType\":\"uint16\",\"name\":\"emitterChainId\",\"type\":\"uint16\"},{\"internalType\":\"bytes32\",\"name\":\"emitterAddress\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"sequence\",\"type\":\"uint64\"},{\"internalType\":\"uint8\",\"name\":\"consistencyLevel\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"guardianSetIndex\",\"type\":\"uint32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"guardianIndex\",\"type\":\"uint8\"}],\"internalType\":\"structStructs.Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"}],\"internalType\":\"structStructs.VM\",\"name\":\"vm\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_vm\",\"type\":\"bytes\"}],\"name\":\"submitContractUpgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_vm\",\"type\":\"bytes\"}],\"name\":\"submitNewGuardianSet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_vm\",\"type\":\"bytes\"}],\"name\":\"submitSetMessageFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_vm\",\"type\":\"bytes\"}],\"name\":\"submitTransferFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"guardianIndex\",\"type\":\"uint8\"}],\"internalType\":\"structStructs.Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expirationTime\",\"type\":\"uint32\"}],\"internalType\":\"structStructs.GuardianSet\",\"name\":\"guardianSet\",\"type\":\"tuple\"}],\"name\":\"verifySignatures\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"valid\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"internalType\":\"uint16\",\"name\":\"emitterChainId\",\"type\":\"uint16\"},{\"internalType\":\"bytes32\",\"name\":\"emitterAddress\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"sequence\",\"type\":\"uint64\"},{\"internalType\":\"uint8\",\"name\":\"consistencyLevel\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"guardianSetIndex\",\"type\":\"uint32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"guardianIndex\",\"type\":\"uint8\"}],\"internalType\":\"structStructs.Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"}],\"internalType\":\"structStructs.VM\",\"name\":\"vm\",\"type\":\"tuple\"}],\"name\":\"verifyVM\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"valid\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"payload\",\"type\":\"bytes\"},{\"internalType\":\"uint8\",\"name\":\"consistencyLevel\",\"type\":\"uint8\"}],\"name\":\"publishMessage\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"sequence\",\"type\":\"uint64\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"initialGuardians\",\"type\":\"address[]\"},{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"governanceChainId\",\"type\":\"uint16\"},{\"internalType\":\"bytes32\",\"name\":\"governanceContract\",\"type\":\"bytes32\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Abi is an auto generated Go binding around an Ethereum contract.
type Abi struct {
	AbiCaller     // Read-only binding to the contract
	AbiTransactor // Write-only binding to the contract
	AbiFilterer   // Log filterer for contract events
}

// AbiCaller is an auto generated read-only Go binding around an Ethereum contract.
type AbiCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AbiTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AbiFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AbiSession struct {
	Contract     *Abi              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AbiCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AbiCallerSession struct {
	Contract *AbiCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// AbiTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AbiTransactorSession struct {
	Contract     *AbiTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AbiRaw is an auto generated low-level Go binding around an Ethereum contract.
type AbiRaw struct {
	Contract *Abi // Generic contract binding to access the raw methods on
}

// AbiCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AbiCallerRaw struct {
	Contract *AbiCaller // Generic read-only contract binding to access the raw methods on
}

// AbiTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AbiTransactorRaw struct {
	Contract *AbiTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAbi creates a new instance of Abi, bound to a specific deployed contract.
func NewAbi(address common.Address, backend bind.ContractBackend) (*Abi, error) {
	contract, err := bindAbi(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Abi{AbiCaller: AbiCaller{contract: contract}, AbiTransactor: AbiTransactor{contract: contract}, AbiFilterer: AbiFilterer{contract: contract}}, nil
}

// NewAbiCaller creates a new read-only instance of Abi, bound to a specific deployed contract.
func NewAbiCaller(address common.Address, caller bind.ContractCaller) (*AbiCaller, error) {
	contract, err := bindAbi(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AbiCaller{contract: contract}, nil
}

// NewAbiTransactor creates a new write-only instance of Abi, bound to a specific deployed contract.
func NewAbiTransactor(address common.Address, transactor bind.ContractTransactor) (*AbiTransactor, error) {
	contract, err := bindAbi(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AbiTransactor{contract: contract}, nil
}

// NewAbiFilterer creates a new log filterer instance of Abi, bound to a specific deployed contract.
func NewAbiFilterer(address common.Address, filterer bind.ContractFilterer) (*AbiFilterer, error) {
	contract, err := bindAbi(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AbiFilterer{contract: contract}, nil
}

// bindAbi binds a generic wrapper to an already deployed contract.
func bindAbi(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AbiABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Abi *AbiRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Abi.Contract.AbiCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Abi *AbiRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Abi.Contract.AbiTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Abi *AbiRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Abi.Contract.AbiTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Abi *AbiCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Abi.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Abi *AbiTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Abi.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Abi *AbiTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Abi.Contract.contract.Transact(opts, method, params...)
}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint16)
func (_Abi *AbiCaller) ChainId(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "chainId")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint16)
func (_Abi *AbiSession) ChainId() (uint16, error) {
	return _Abi.Contract.ChainId(&_Abi.CallOpts)
}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint16)
func (_Abi *AbiCallerSession) ChainId() (uint16, error) {
	return _Abi.Contract.ChainId(&_Abi.CallOpts)
}

// GetCurrentGuardianSetIndex is a free data retrieval call binding the contract method 0x1cfe7951.
//
// Solidity: function getCurrentGuardianSetIndex() view returns(uint32)
func (_Abi *AbiCaller) GetCurrentGuardianSetIndex(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "getCurrentGuardianSetIndex")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// GetCurrentGuardianSetIndex is a free data retrieval call binding the contract method 0x1cfe7951.
//
// Solidity: function getCurrentGuardianSetIndex() view returns(uint32)
func (_Abi *AbiSession) GetCurrentGuardianSetIndex() (uint32, error) {
	return _Abi.Contract.GetCurrentGuardianSetIndex(&_Abi.CallOpts)
}

// GetCurrentGuardianSetIndex is a free data retrieval call binding the contract method 0x1cfe7951.
//
// Solidity: function getCurrentGuardianSetIndex() view returns(uint32)
func (_Abi *AbiCallerSession) GetCurrentGuardianSetIndex() (uint32, error) {
	return _Abi.Contract.GetCurrentGuardianSetIndex(&_Abi.CallOpts)
}

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 index) view returns((address[],uint32))
func (_Abi *AbiCaller) GetGuardianSet(opts *bind.CallOpts, index uint32) (StructsGuardianSet, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "getGuardianSet", index)

	if err != nil {
		return *new(StructsGuardianSet), err
	}

	out0 := *abi.ConvertType(out[0], new(StructsGuardianSet)).(*StructsGuardianSet)

	return out0, err

}

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 index) view returns((address[],uint32))
func (_Abi *AbiSession) GetGuardianSet(index uint32) (StructsGuardianSet, error) {
	return _Abi.Contract.GetGuardianSet(&_Abi.CallOpts, index)
}

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 index) view returns((address[],uint32))
func (_Abi *AbiCallerSession) GetGuardianSet(index uint32) (StructsGuardianSet, error) {
	return _Abi.Contract.GetGuardianSet(&_Abi.CallOpts, index)
}

// GetGuardianSetExpiry is a free data retrieval call binding the contract method 0xeb8d3f12.
//
// Solidity: function getGuardianSetExpiry() view returns(uint32)
func (_Abi *AbiCaller) GetGuardianSetExpiry(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "getGuardianSetExpiry")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// GetGuardianSetExpiry is a free data retrieval call binding the contract method 0xeb8d3f12.
//
// Solidity: function getGuardianSetExpiry() view returns(uint32)
func (_Abi *AbiSession) GetGuardianSetExpiry() (uint32, error) {
	return _Abi.Contract.GetGuardianSetExpiry(&_Abi.CallOpts)
}

// GetGuardianSetExpiry is a free data retrieval call binding the contract method 0xeb8d3f12.
//
// Solidity: function getGuardianSetExpiry() view returns(uint32)
func (_Abi *AbiCallerSession) GetGuardianSetExpiry() (uint32, error) {
	return _Abi.Contract.GetGuardianSetExpiry(&_Abi.CallOpts)
}

// GovernanceActionIsConsumed is a free data retrieval call binding the contract method 0x2c3c02a4.
//
// Solidity: function governanceActionIsConsumed(bytes32 hash) view returns(bool)
func (_Abi *AbiCaller) GovernanceActionIsConsumed(opts *bind.CallOpts, hash [32]byte) (bool, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "governanceActionIsConsumed", hash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GovernanceActionIsConsumed is a free data retrieval call binding the contract method 0x2c3c02a4.
//
// Solidity: function governanceActionIsConsumed(bytes32 hash) view returns(bool)
func (_Abi *AbiSession) GovernanceActionIsConsumed(hash [32]byte) (bool, error) {
	return _Abi.Contract.GovernanceActionIsConsumed(&_Abi.CallOpts, hash)
}

// GovernanceActionIsConsumed is a free data retrieval call binding the contract method 0x2c3c02a4.
//
// Solidity: function governanceActionIsConsumed(bytes32 hash) view returns(bool)
func (_Abi *AbiCallerSession) GovernanceActionIsConsumed(hash [32]byte) (bool, error) {
	return _Abi.Contract.GovernanceActionIsConsumed(&_Abi.CallOpts, hash)
}

// GovernanceChainId is a free data retrieval call binding the contract method 0xfbe3c2cd.
//
// Solidity: function governanceChainId() view returns(uint16)
func (_Abi *AbiCaller) GovernanceChainId(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "governanceChainId")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// GovernanceChainId is a free data retrieval call binding the contract method 0xfbe3c2cd.
//
// Solidity: function governanceChainId() view returns(uint16)
func (_Abi *AbiSession) GovernanceChainId() (uint16, error) {
	return _Abi.Contract.GovernanceChainId(&_Abi.CallOpts)
}

// GovernanceChainId is a free data retrieval call binding the contract method 0xfbe3c2cd.
//
// Solidity: function governanceChainId() view returns(uint16)
func (_Abi *AbiCallerSession) GovernanceChainId() (uint16, error) {
	return _Abi.Contract.GovernanceChainId(&_Abi.CallOpts)
}

// GovernanceContract is a free data retrieval call binding the contract method 0xb172b222.
//
// Solidity: function governanceContract() view returns(bytes32)
func (_Abi *AbiCaller) GovernanceContract(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "governanceContract")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GovernanceContract is a free data retrieval call binding the contract method 0xb172b222.
//
// Solidity: function governanceContract() view returns(bytes32)
func (_Abi *AbiSession) GovernanceContract() ([32]byte, error) {
	return _Abi.Contract.GovernanceContract(&_Abi.CallOpts)
}

// GovernanceContract is a free data retrieval call binding the contract method 0xb172b222.
//
// Solidity: function governanceContract() view returns(bytes32)
func (_Abi *AbiCallerSession) GovernanceContract() ([32]byte, error) {
	return _Abi.Contract.GovernanceContract(&_Abi.CallOpts)
}

// IsInitialized is a free data retrieval call binding the contract method 0xd60b347f.
//
// Solidity: function isInitialized(address impl) view returns(bool)
func (_Abi *AbiCaller) IsInitialized(opts *bind.CallOpts, impl common.Address) (bool, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "isInitialized", impl)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsInitialized is a free data retrieval call binding the contract method 0xd60b347f.
//
// Solidity: function isInitialized(address impl) view returns(bool)
func (_Abi *AbiSession) IsInitialized(impl common.Address) (bool, error) {
	return _Abi.Contract.IsInitialized(&_Abi.CallOpts, impl)
}

// IsInitialized is a free data retrieval call binding the contract method 0xd60b347f.
//
// Solidity: function isInitialized(address impl) view returns(bool)
func (_Abi *AbiCallerSession) IsInitialized(impl common.Address) (bool, error) {
	return _Abi.Contract.IsInitialized(&_Abi.CallOpts, impl)
}

// MessageFee is a free data retrieval call binding the contract method 0x1a90a219.
//
// Solidity: function messageFee() view returns(uint256)
func (_Abi *AbiCaller) MessageFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "messageFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageFee is a free data retrieval call binding the contract method 0x1a90a219.
//
// Solidity: function messageFee() view returns(uint256)
func (_Abi *AbiSession) MessageFee() (*big.Int, error) {
	return _Abi.Contract.MessageFee(&_Abi.CallOpts)
}

// MessageFee is a free data retrieval call binding the contract method 0x1a90a219.
//
// Solidity: function messageFee() view returns(uint256)
func (_Abi *AbiCallerSession) MessageFee() (*big.Int, error) {
	return _Abi.Contract.MessageFee(&_Abi.CallOpts)
}

// NextSequence is a free data retrieval call binding the contract method 0x4cf842b5.
//
// Solidity: function nextSequence(address emitter) view returns(uint64)
func (_Abi *AbiCaller) NextSequence(opts *bind.CallOpts, emitter common.Address) (uint64, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "nextSequence", emitter)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// NextSequence is a free data retrieval call binding the contract method 0x4cf842b5.
//
// Solidity: function nextSequence(address emitter) view returns(uint64)
func (_Abi *AbiSession) NextSequence(emitter common.Address) (uint64, error) {
	return _Abi.Contract.NextSequence(&_Abi.CallOpts, emitter)
}

// NextSequence is a free data retrieval call binding the contract method 0x4cf842b5.
//
// Solidity: function nextSequence(address emitter) view returns(uint64)
func (_Abi *AbiCallerSession) NextSequence(emitter common.Address) (uint64, error) {
	return _Abi.Contract.NextSequence(&_Abi.CallOpts, emitter)
}

// ParseAndVerifyVM is a free data retrieval call binding the contract method 0xc0fd8bde.
//
// Solidity: function parseAndVerifyVM(bytes encodedVM) view returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm, bool valid, string reason)
func (_Abi *AbiCaller) ParseAndVerifyVM(opts *bind.CallOpts, encodedVM []byte) (struct {
	Vm     StructsVM
	Valid  bool
	Reason string
}, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseAndVerifyVM", encodedVM)

	outstruct := new(struct {
		Vm     StructsVM
		Valid  bool
		Reason string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Vm = *abi.ConvertType(out[0], new(StructsVM)).(*StructsVM)
	outstruct.Valid = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.Reason = *abi.ConvertType(out[2], new(string)).(*string)

	return *outstruct, err

}

// ParseAndVerifyVM is a free data retrieval call binding the contract method 0xc0fd8bde.
//
// Solidity: function parseAndVerifyVM(bytes encodedVM) view returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm, bool valid, string reason)
func (_Abi *AbiSession) ParseAndVerifyVM(encodedVM []byte) (struct {
	Vm     StructsVM
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.ParseAndVerifyVM(&_Abi.CallOpts, encodedVM)
}

// ParseAndVerifyVM is a free data retrieval call binding the contract method 0xc0fd8bde.
//
// Solidity: function parseAndVerifyVM(bytes encodedVM) view returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm, bool valid, string reason)
func (_Abi *AbiCallerSession) ParseAndVerifyVM(encodedVM []byte) (struct {
	Vm     StructsVM
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.ParseAndVerifyVM(&_Abi.CallOpts, encodedVM)
}

// ParseContractUpgrade is a free data retrieval call binding the contract method 0x4fdc60fa.
//
// Solidity: function parseContractUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,address) cu)
func (_Abi *AbiCaller) ParseContractUpgrade(opts *bind.CallOpts, encodedUpgrade []byte) (GovernanceStructsContractUpgrade, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseContractUpgrade", encodedUpgrade)

	if err != nil {
		return *new(GovernanceStructsContractUpgrade), err
	}

	out0 := *abi.ConvertType(out[0], new(GovernanceStructsContractUpgrade)).(*GovernanceStructsContractUpgrade)

	return out0, err

}

// ParseContractUpgrade is a free data retrieval call binding the contract method 0x4fdc60fa.
//
// Solidity: function parseContractUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,address) cu)
func (_Abi *AbiSession) ParseContractUpgrade(encodedUpgrade []byte) (GovernanceStructsContractUpgrade, error) {
	return _Abi.Contract.ParseContractUpgrade(&_Abi.CallOpts, encodedUpgrade)
}

// ParseContractUpgrade is a free data retrieval call binding the contract method 0x4fdc60fa.
//
// Solidity: function parseContractUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,address) cu)
func (_Abi *AbiCallerSession) ParseContractUpgrade(encodedUpgrade []byte) (GovernanceStructsContractUpgrade, error) {
	return _Abi.Contract.ParseContractUpgrade(&_Abi.CallOpts, encodedUpgrade)
}

// ParseGuardianSetUpgrade is a free data retrieval call binding the contract method 0x04ca84cf.
//
// Solidity: function parseGuardianSetUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,(address[],uint32),uint32) gsu)
func (_Abi *AbiCaller) ParseGuardianSetUpgrade(opts *bind.CallOpts, encodedUpgrade []byte) (GovernanceStructsGuardianSetUpgrade, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseGuardianSetUpgrade", encodedUpgrade)

	if err != nil {
		return *new(GovernanceStructsGuardianSetUpgrade), err
	}

	out0 := *abi.ConvertType(out[0], new(GovernanceStructsGuardianSetUpgrade)).(*GovernanceStructsGuardianSetUpgrade)

	return out0, err

}

// ParseGuardianSetUpgrade is a free data retrieval call binding the contract method 0x04ca84cf.
//
// Solidity: function parseGuardianSetUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,(address[],uint32),uint32) gsu)
func (_Abi *AbiSession) ParseGuardianSetUpgrade(encodedUpgrade []byte) (GovernanceStructsGuardianSetUpgrade, error) {
	return _Abi.Contract.ParseGuardianSetUpgrade(&_Abi.CallOpts, encodedUpgrade)
}

// ParseGuardianSetUpgrade is a free data retrieval call binding the contract method 0x04ca84cf.
//
// Solidity: function parseGuardianSetUpgrade(bytes encodedUpgrade) pure returns((bytes32,uint8,uint16,(address[],uint32),uint32) gsu)
func (_Abi *AbiCallerSession) ParseGuardianSetUpgrade(encodedUpgrade []byte) (GovernanceStructsGuardianSetUpgrade, error) {
	return _Abi.Contract.ParseGuardianSetUpgrade(&_Abi.CallOpts, encodedUpgrade)
}

// ParseSetMessageFee is a free data retrieval call binding the contract method 0x515f3247.
//
// Solidity: function parseSetMessageFee(bytes encodedSetMessageFee) pure returns((bytes32,uint8,uint16,uint256) smf)
func (_Abi *AbiCaller) ParseSetMessageFee(opts *bind.CallOpts, encodedSetMessageFee []byte) (GovernanceStructsSetMessageFee, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseSetMessageFee", encodedSetMessageFee)

	if err != nil {
		return *new(GovernanceStructsSetMessageFee), err
	}

	out0 := *abi.ConvertType(out[0], new(GovernanceStructsSetMessageFee)).(*GovernanceStructsSetMessageFee)

	return out0, err

}

// ParseSetMessageFee is a free data retrieval call binding the contract method 0x515f3247.
//
// Solidity: function parseSetMessageFee(bytes encodedSetMessageFee) pure returns((bytes32,uint8,uint16,uint256) smf)
func (_Abi *AbiSession) ParseSetMessageFee(encodedSetMessageFee []byte) (GovernanceStructsSetMessageFee, error) {
	return _Abi.Contract.ParseSetMessageFee(&_Abi.CallOpts, encodedSetMessageFee)
}

// ParseSetMessageFee is a free data retrieval call binding the contract method 0x515f3247.
//
// Solidity: function parseSetMessageFee(bytes encodedSetMessageFee) pure returns((bytes32,uint8,uint16,uint256) smf)
func (_Abi *AbiCallerSession) ParseSetMessageFee(encodedSetMessageFee []byte) (GovernanceStructsSetMessageFee, error) {
	return _Abi.Contract.ParseSetMessageFee(&_Abi.CallOpts, encodedSetMessageFee)
}

// ParseTransferFees is a free data retrieval call binding the contract method 0x0319e59c.
//
// Solidity: function parseTransferFees(bytes encodedTransferFees) pure returns((bytes32,uint8,uint16,uint256,bytes32) tf)
func (_Abi *AbiCaller) ParseTransferFees(opts *bind.CallOpts, encodedTransferFees []byte) (GovernanceStructsTransferFees, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseTransferFees", encodedTransferFees)

	if err != nil {
		return *new(GovernanceStructsTransferFees), err
	}

	out0 := *abi.ConvertType(out[0], new(GovernanceStructsTransferFees)).(*GovernanceStructsTransferFees)

	return out0, err

}

// ParseTransferFees is a free data retrieval call binding the contract method 0x0319e59c.
//
// Solidity: function parseTransferFees(bytes encodedTransferFees) pure returns((bytes32,uint8,uint16,uint256,bytes32) tf)
func (_Abi *AbiSession) ParseTransferFees(encodedTransferFees []byte) (GovernanceStructsTransferFees, error) {
	return _Abi.Contract.ParseTransferFees(&_Abi.CallOpts, encodedTransferFees)
}

// ParseTransferFees is a free data retrieval call binding the contract method 0x0319e59c.
//
// Solidity: function parseTransferFees(bytes encodedTransferFees) pure returns((bytes32,uint8,uint16,uint256,bytes32) tf)
func (_Abi *AbiCallerSession) ParseTransferFees(encodedTransferFees []byte) (GovernanceStructsTransferFees, error) {
	return _Abi.Contract.ParseTransferFees(&_Abi.CallOpts, encodedTransferFees)
}

// ParseVM is a free data retrieval call binding the contract method 0xa9e11893.
//
// Solidity: function parseVM(bytes encodedVM) pure returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm)
func (_Abi *AbiCaller) ParseVM(opts *bind.CallOpts, encodedVM []byte) (StructsVM, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "parseVM", encodedVM)

	if err != nil {
		return *new(StructsVM), err
	}

	out0 := *abi.ConvertType(out[0], new(StructsVM)).(*StructsVM)

	return out0, err

}

// ParseVM is a free data retrieval call binding the contract method 0xa9e11893.
//
// Solidity: function parseVM(bytes encodedVM) pure returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm)
func (_Abi *AbiSession) ParseVM(encodedVM []byte) (StructsVM, error) {
	return _Abi.Contract.ParseVM(&_Abi.CallOpts, encodedVM)
}

// ParseVM is a free data retrieval call binding the contract method 0xa9e11893.
//
// Solidity: function parseVM(bytes encodedVM) pure returns((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm)
func (_Abi *AbiCallerSession) ParseVM(encodedVM []byte) (StructsVM, error) {
	return _Abi.Contract.ParseVM(&_Abi.CallOpts, encodedVM)
}

// VerifySignatures is a free data retrieval call binding the contract method 0xa0cce1b3.
//
// Solidity: function verifySignatures(bytes32 hash, (bytes32,bytes32,uint8,uint8)[] signatures, (address[],uint32) guardianSet) pure returns(bool valid, string reason)
func (_Abi *AbiCaller) VerifySignatures(opts *bind.CallOpts, hash [32]byte, signatures []StructsSignature, guardianSet StructsGuardianSet) (struct {
	Valid  bool
	Reason string
}, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "verifySignatures", hash, signatures, guardianSet)

	outstruct := new(struct {
		Valid  bool
		Reason string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Valid = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.Reason = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// VerifySignatures is a free data retrieval call binding the contract method 0xa0cce1b3.
//
// Solidity: function verifySignatures(bytes32 hash, (bytes32,bytes32,uint8,uint8)[] signatures, (address[],uint32) guardianSet) pure returns(bool valid, string reason)
func (_Abi *AbiSession) VerifySignatures(hash [32]byte, signatures []StructsSignature, guardianSet StructsGuardianSet) (struct {
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.VerifySignatures(&_Abi.CallOpts, hash, signatures, guardianSet)
}

// VerifySignatures is a free data retrieval call binding the contract method 0xa0cce1b3.
//
// Solidity: function verifySignatures(bytes32 hash, (bytes32,bytes32,uint8,uint8)[] signatures, (address[],uint32) guardianSet) pure returns(bool valid, string reason)
func (_Abi *AbiCallerSession) VerifySignatures(hash [32]byte, signatures []StructsSignature, guardianSet StructsGuardianSet) (struct {
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.VerifySignatures(&_Abi.CallOpts, hash, signatures, guardianSet)
}

// VerifyVM is a free data retrieval call binding the contract method 0x875be02a.
//
// Solidity: function verifyVM((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm) view returns(bool valid, string reason)
func (_Abi *AbiCaller) VerifyVM(opts *bind.CallOpts, vm StructsVM) (struct {
	Valid  bool
	Reason string
}, error) {
	var out []interface{}
	err := _Abi.contract.Call(opts, &out, "verifyVM", vm)

	outstruct := new(struct {
		Valid  bool
		Reason string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Valid = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.Reason = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// VerifyVM is a free data retrieval call binding the contract method 0x875be02a.
//
// Solidity: function verifyVM((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm) view returns(bool valid, string reason)
func (_Abi *AbiSession) VerifyVM(vm StructsVM) (struct {
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.VerifyVM(&_Abi.CallOpts, vm)
}

// VerifyVM is a free data retrieval call binding the contract method 0x875be02a.
//
// Solidity: function verifyVM((uint8,uint32,uint32,uint16,bytes32,uint64,uint8,bytes,uint32,(bytes32,bytes32,uint8,uint8)[],bytes32) vm) view returns(bool valid, string reason)
func (_Abi *AbiCallerSession) VerifyVM(vm StructsVM) (struct {
	Valid  bool
	Reason string
}, error) {
	return _Abi.Contract.VerifyVM(&_Abi.CallOpts, vm)
}

// Initialize is a paid mutator transaction binding the contract method 0xf6079017.
//
// Solidity: function initialize(address[] initialGuardians, uint16 chainId, uint16 governanceChainId, bytes32 governanceContract) returns()
func (_Abi *AbiTransactor) Initialize(opts *bind.TransactOpts, initialGuardians []common.Address, chainId uint16, governanceChainId uint16, governanceContract [32]byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "initialize", initialGuardians, chainId, governanceChainId, governanceContract)
}

// Initialize is a paid mutator transaction binding the contract method 0xf6079017.
//
// Solidity: function initialize(address[] initialGuardians, uint16 chainId, uint16 governanceChainId, bytes32 governanceContract) returns()
func (_Abi *AbiSession) Initialize(initialGuardians []common.Address, chainId uint16, governanceChainId uint16, governanceContract [32]byte) (*types.Transaction, error) {
	return _Abi.Contract.Initialize(&_Abi.TransactOpts, initialGuardians, chainId, governanceChainId, governanceContract)
}

// Initialize is a paid mutator transaction binding the contract method 0xf6079017.
//
// Solidity: function initialize(address[] initialGuardians, uint16 chainId, uint16 governanceChainId, bytes32 governanceContract) returns()
func (_Abi *AbiTransactorSession) Initialize(initialGuardians []common.Address, chainId uint16, governanceChainId uint16, governanceContract [32]byte) (*types.Transaction, error) {
	return _Abi.Contract.Initialize(&_Abi.TransactOpts, initialGuardians, chainId, governanceChainId, governanceContract)
}

// PublishMessage is a paid mutator transaction binding the contract method 0xb19a437e.
//
// Solidity: function publishMessage(uint32 nonce, bytes payload, uint8 consistencyLevel) payable returns(uint64 sequence)
func (_Abi *AbiTransactor) PublishMessage(opts *bind.TransactOpts, nonce uint32, payload []byte, consistencyLevel uint8) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "publishMessage", nonce, payload, consistencyLevel)
}

// PublishMessage is a paid mutator transaction binding the contract method 0xb19a437e.
//
// Solidity: function publishMessage(uint32 nonce, bytes payload, uint8 consistencyLevel) payable returns(uint64 sequence)
func (_Abi *AbiSession) PublishMessage(nonce uint32, payload []byte, consistencyLevel uint8) (*types.Transaction, error) {
	return _Abi.Contract.PublishMessage(&_Abi.TransactOpts, nonce, payload, consistencyLevel)
}

// PublishMessage is a paid mutator transaction binding the contract method 0xb19a437e.
//
// Solidity: function publishMessage(uint32 nonce, bytes payload, uint8 consistencyLevel) payable returns(uint64 sequence)
func (_Abi *AbiTransactorSession) PublishMessage(nonce uint32, payload []byte, consistencyLevel uint8) (*types.Transaction, error) {
	return _Abi.Contract.PublishMessage(&_Abi.TransactOpts, nonce, payload, consistencyLevel)
}

// SubmitContractUpgrade is a paid mutator transaction binding the contract method 0x5cb8cae2.
//
// Solidity: function submitContractUpgrade(bytes _vm) returns()
func (_Abi *AbiTransactor) SubmitContractUpgrade(opts *bind.TransactOpts, _vm []byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "submitContractUpgrade", _vm)
}

// SubmitContractUpgrade is a paid mutator transaction binding the contract method 0x5cb8cae2.
//
// Solidity: function submitContractUpgrade(bytes _vm) returns()
func (_Abi *AbiSession) SubmitContractUpgrade(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitContractUpgrade(&_Abi.TransactOpts, _vm)
}

// SubmitContractUpgrade is a paid mutator transaction binding the contract method 0x5cb8cae2.
//
// Solidity: function submitContractUpgrade(bytes _vm) returns()
func (_Abi *AbiTransactorSession) SubmitContractUpgrade(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitContractUpgrade(&_Abi.TransactOpts, _vm)
}

// SubmitNewGuardianSet is a paid mutator transaction binding the contract method 0x6606b4e0.
//
// Solidity: function submitNewGuardianSet(bytes _vm) returns()
func (_Abi *AbiTransactor) SubmitNewGuardianSet(opts *bind.TransactOpts, _vm []byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "submitNewGuardianSet", _vm)
}

// SubmitNewGuardianSet is a paid mutator transaction binding the contract method 0x6606b4e0.
//
// Solidity: function submitNewGuardianSet(bytes _vm) returns()
func (_Abi *AbiSession) SubmitNewGuardianSet(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitNewGuardianSet(&_Abi.TransactOpts, _vm)
}

// SubmitNewGuardianSet is a paid mutator transaction binding the contract method 0x6606b4e0.
//
// Solidity: function submitNewGuardianSet(bytes _vm) returns()
func (_Abi *AbiTransactorSession) SubmitNewGuardianSet(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitNewGuardianSet(&_Abi.TransactOpts, _vm)
}

// SubmitSetMessageFee is a paid mutator transaction binding the contract method 0xf42bc641.
//
// Solidity: function submitSetMessageFee(bytes _vm) returns()
func (_Abi *AbiTransactor) SubmitSetMessageFee(opts *bind.TransactOpts, _vm []byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "submitSetMessageFee", _vm)
}

// SubmitSetMessageFee is a paid mutator transaction binding the contract method 0xf42bc641.
//
// Solidity: function submitSetMessageFee(bytes _vm) returns()
func (_Abi *AbiSession) SubmitSetMessageFee(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitSetMessageFee(&_Abi.TransactOpts, _vm)
}

// SubmitSetMessageFee is a paid mutator transaction binding the contract method 0xf42bc641.
//
// Solidity: function submitSetMessageFee(bytes _vm) returns()
func (_Abi *AbiTransactorSession) SubmitSetMessageFee(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitSetMessageFee(&_Abi.TransactOpts, _vm)
}

// SubmitTransferFees is a paid mutator transaction binding the contract method 0x93df337e.
//
// Solidity: function submitTransferFees(bytes _vm) returns()
func (_Abi *AbiTransactor) SubmitTransferFees(opts *bind.TransactOpts, _vm []byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "submitTransferFees", _vm)
}

// SubmitTransferFees is a paid mutator transaction binding the contract method 0x93df337e.
//
// Solidity: function submitTransferFees(bytes _vm) returns()
func (_Abi *AbiSession) SubmitTransferFees(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitTransferFees(&_Abi.TransactOpts, _vm)
}

// SubmitTransferFees is a paid mutator transaction binding the contract method 0x93df337e.
//
// Solidity: function submitTransferFees(bytes _vm) returns()
func (_Abi *AbiTransactorSession) SubmitTransferFees(_vm []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitTransferFees(&_Abi.TransactOpts, _vm)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Abi *AbiTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _Abi.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Abi *AbiSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Abi.Contract.Fallback(&_Abi.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Abi *AbiTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Abi.Contract.Fallback(&_Abi.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Abi *AbiTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Abi.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Abi *AbiSession) Receive() (*types.Transaction, error) {
	return _Abi.Contract.Receive(&_Abi.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Abi *AbiTransactorSession) Receive() (*types.Transaction, error) {
	return _Abi.Contract.Receive(&_Abi.TransactOpts)
}

// AbiAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the Abi contract.
type AbiAdminChangedIterator struct {
	Event *AbiAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiAdminChanged represents a AdminChanged event raised by the Abi contract.
type AbiAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_Abi *AbiFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*AbiAdminChangedIterator, error) {

	logs, sub, err := _Abi.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &AbiAdminChangedIterator{contract: _Abi.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_Abi *AbiFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *AbiAdminChanged) (event.Subscription, error) {

	logs, sub, err := _Abi.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiAdminChanged)
				if err := _Abi.contract.UnpackLog(event, "AdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_Abi *AbiFilterer) ParseAdminChanged(log types.Log) (*AbiAdminChanged, error) {
	event := new(AbiAdminChanged)
	if err := _Abi.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbiBeaconUpgradedIterator is returned from FilterBeaconUpgraded and is used to iterate over the raw logs and unpacked data for BeaconUpgraded events raised by the Abi contract.
type AbiBeaconUpgradedIterator struct {
	Event *AbiBeaconUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiBeaconUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiBeaconUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiBeaconUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiBeaconUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiBeaconUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiBeaconUpgraded represents a BeaconUpgraded event raised by the Abi contract.
type AbiBeaconUpgraded struct {
	Beacon common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBeaconUpgraded is a free log retrieval operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_Abi *AbiFilterer) FilterBeaconUpgraded(opts *bind.FilterOpts, beacon []common.Address) (*AbiBeaconUpgradedIterator, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return &AbiBeaconUpgradedIterator{contract: _Abi.contract, event: "BeaconUpgraded", logs: logs, sub: sub}, nil
}

// WatchBeaconUpgraded is a free log subscription operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_Abi *AbiFilterer) WatchBeaconUpgraded(opts *bind.WatchOpts, sink chan<- *AbiBeaconUpgraded, beacon []common.Address) (event.Subscription, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiBeaconUpgraded)
				if err := _Abi.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBeaconUpgraded is a log parse operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_Abi *AbiFilterer) ParseBeaconUpgraded(log types.Log) (*AbiBeaconUpgraded, error) {
	event := new(AbiBeaconUpgraded)
	if err := _Abi.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbiContractUpgradedIterator is returned from FilterContractUpgraded and is used to iterate over the raw logs and unpacked data for ContractUpgraded events raised by the Abi contract.
type AbiContractUpgradedIterator struct {
	Event *AbiContractUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiContractUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiContractUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiContractUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiContractUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiContractUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiContractUpgraded represents a ContractUpgraded event raised by the Abi contract.
type AbiContractUpgraded struct {
	OldContract common.Address
	NewContract common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterContractUpgraded is a free log retrieval operation binding the contract event 0x2e4cc16c100f0b55e2df82ab0b1a7e294aa9cbd01b48fbaf622683fbc0507a49.
//
// Solidity: event ContractUpgraded(address indexed oldContract, address indexed newContract)
func (_Abi *AbiFilterer) FilterContractUpgraded(opts *bind.FilterOpts, oldContract []common.Address, newContract []common.Address) (*AbiContractUpgradedIterator, error) {

	var oldContractRule []interface{}
	for _, oldContractItem := range oldContract {
		oldContractRule = append(oldContractRule, oldContractItem)
	}
	var newContractRule []interface{}
	for _, newContractItem := range newContract {
		newContractRule = append(newContractRule, newContractItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "ContractUpgraded", oldContractRule, newContractRule)
	if err != nil {
		return nil, err
	}
	return &AbiContractUpgradedIterator{contract: _Abi.contract, event: "ContractUpgraded", logs: logs, sub: sub}, nil
}

// WatchContractUpgraded is a free log subscription operation binding the contract event 0x2e4cc16c100f0b55e2df82ab0b1a7e294aa9cbd01b48fbaf622683fbc0507a49.
//
// Solidity: event ContractUpgraded(address indexed oldContract, address indexed newContract)
func (_Abi *AbiFilterer) WatchContractUpgraded(opts *bind.WatchOpts, sink chan<- *AbiContractUpgraded, oldContract []common.Address, newContract []common.Address) (event.Subscription, error) {

	var oldContractRule []interface{}
	for _, oldContractItem := range oldContract {
		oldContractRule = append(oldContractRule, oldContractItem)
	}
	var newContractRule []interface{}
	for _, newContractItem := range newContract {
		newContractRule = append(newContractRule, newContractItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "ContractUpgraded", oldContractRule, newContractRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiContractUpgraded)
				if err := _Abi.contract.UnpackLog(event, "ContractUpgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseContractUpgraded is a log parse operation binding the contract event 0x2e4cc16c100f0b55e2df82ab0b1a7e294aa9cbd01b48fbaf622683fbc0507a49.
//
// Solidity: event ContractUpgraded(address indexed oldContract, address indexed newContract)
func (_Abi *AbiFilterer) ParseContractUpgraded(log types.Log) (*AbiContractUpgraded, error) {
	event := new(AbiContractUpgraded)
	if err := _Abi.contract.UnpackLog(event, "ContractUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbiGuardianSetAddedIterator is returned from FilterGuardianSetAdded and is used to iterate over the raw logs and unpacked data for GuardianSetAdded events raised by the Abi contract.
type AbiGuardianSetAddedIterator struct {
	Event *AbiGuardianSetAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiGuardianSetAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiGuardianSetAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiGuardianSetAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiGuardianSetAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiGuardianSetAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiGuardianSetAdded represents a GuardianSetAdded event raised by the Abi contract.
type AbiGuardianSetAdded struct {
	Index uint32
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGuardianSetAdded is a free log retrieval operation binding the contract event 0x2384dbc52f7b617fb7c5aa71e5455a21ff21d58604bb6daef6af2bb44aadebdd.
//
// Solidity: event GuardianSetAdded(uint32 indexed index)
func (_Abi *AbiFilterer) FilterGuardianSetAdded(opts *bind.FilterOpts, index []uint32) (*AbiGuardianSetAddedIterator, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "GuardianSetAdded", indexRule)
	if err != nil {
		return nil, err
	}
	return &AbiGuardianSetAddedIterator{contract: _Abi.contract, event: "GuardianSetAdded", logs: logs, sub: sub}, nil
}

// WatchGuardianSetAdded is a free log subscription operation binding the contract event 0x2384dbc52f7b617fb7c5aa71e5455a21ff21d58604bb6daef6af2bb44aadebdd.
//
// Solidity: event GuardianSetAdded(uint32 indexed index)
func (_Abi *AbiFilterer) WatchGuardianSetAdded(opts *bind.WatchOpts, sink chan<- *AbiGuardianSetAdded, index []uint32) (event.Subscription, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "GuardianSetAdded", indexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiGuardianSetAdded)
				if err := _Abi.contract.UnpackLog(event, "GuardianSetAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseGuardianSetAdded is a log parse operation binding the contract event 0x2384dbc52f7b617fb7c5aa71e5455a21ff21d58604bb6daef6af2bb44aadebdd.
//
// Solidity: event GuardianSetAdded(uint32 indexed index)
func (_Abi *AbiFilterer) ParseGuardianSetAdded(log types.Log) (*AbiGuardianSetAdded, error) {
	event := new(AbiGuardianSetAdded)
	if err := _Abi.contract.UnpackLog(event, "GuardianSetAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbiLogMessagePublishedIterator is returned from FilterLogMessagePublished and is used to iterate over the raw logs and unpacked data for LogMessagePublished events raised by the Abi contract.
type AbiLogMessagePublishedIterator struct {
	Event *AbiLogMessagePublished // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiLogMessagePublishedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiLogMessagePublished)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiLogMessagePublished)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiLogMessagePublishedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiLogMessagePublishedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiLogMessagePublished represents a LogMessagePublished event raised by the Abi contract.
type AbiLogMessagePublished struct {
	Sender           common.Address
	Sequence         uint64
	Nonce            uint32
	Payload          []byte
	ConsistencyLevel uint8
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterLogMessagePublished is a free log retrieval operation binding the contract event 0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2.
//
// Solidity: event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)
func (_Abi *AbiFilterer) FilterLogMessagePublished(opts *bind.FilterOpts, sender []common.Address) (*AbiLogMessagePublishedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "LogMessagePublished", senderRule)
	if err != nil {
		return nil, err
	}
	return &AbiLogMessagePublishedIterator{contract: _Abi.contract, event: "LogMessagePublished", logs: logs, sub: sub}, nil
}

// WatchLogMessagePublished is a free log subscription operation binding the contract event 0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2.
//
// Solidity: event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)
func (_Abi *AbiFilterer) WatchLogMessagePublished(opts *bind.WatchOpts, sink chan<- *AbiLogMessagePublished, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "LogMessagePublished", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiLogMessagePublished)
				if err := _Abi.contract.UnpackLog(event, "LogMessagePublished", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLogMessagePublished is a log parse operation binding the contract event 0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2.
//
// Solidity: event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)
func (_Abi *AbiFilterer) ParseLogMessagePublished(log types.Log) (*AbiLogMessagePublished, error) {
	event := new(AbiLogMessagePublished)
	if err := _Abi.contract.UnpackLog(event, "LogMessagePublished", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbiUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the Abi contract.
type AbiUpgradedIterator struct {
	Event *AbiUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AbiUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AbiUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AbiUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiUpgraded represents a Upgraded event raised by the Abi contract.
type AbiUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Abi *AbiFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*AbiUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &AbiUpgradedIterator{contract: _Abi.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Abi *AbiFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *AbiUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiUpgraded)
				if err := _Abi.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Abi *AbiFilterer) ParseUpgraded(log types.Log) (*AbiUpgraded, error) {
	event := new(AbiUpgraded)
	if err := _Abi.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
