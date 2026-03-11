// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package delegatedmanagersetabi

import (
	"errors"
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
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// DelegatedManagerSetMetaData contains all meta data concerning the DelegatedManagerSet contract.
var DelegatedManagerSetMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"index\",\"type\":\"uint32\"}],\"name\":\"getManagerSet\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// DelegatedManagerSetABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegatedManagerSetMetaData.ABI instead.
var DelegatedManagerSetABI = DelegatedManagerSetMetaData.ABI

// DelegatedManagerSet is an auto generated Go binding around an Ethereum contract.
type DelegatedManagerSet struct {
	DelegatedManagerSetCaller     // Read-only binding to the contract
	DelegatedManagerSetTransactor // Write-only binding to the contract
	DelegatedManagerSetFilterer   // Log filterer for contract events
}

// DelegatedManagerSetCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegatedManagerSetCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedManagerSetTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegatedManagerSetTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedManagerSetFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegatedManagerSetFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedManagerSetSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegatedManagerSetSession struct {
	Contract     *DelegatedManagerSet // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// DelegatedManagerSetCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegatedManagerSetCallerSession struct {
	Contract *DelegatedManagerSetCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// DelegatedManagerSetTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegatedManagerSetTransactorSession struct {
	Contract     *DelegatedManagerSetTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// DelegatedManagerSetRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegatedManagerSetRaw struct {
	Contract *DelegatedManagerSet // Generic contract binding to access the raw methods on
}

// DelegatedManagerSetCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegatedManagerSetCallerRaw struct {
	Contract *DelegatedManagerSetCaller // Generic read-only contract binding to access the raw methods on
}

// DelegatedManagerSetTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegatedManagerSetTransactorRaw struct {
	Contract *DelegatedManagerSetTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegatedManagerSet creates a new instance of DelegatedManagerSet, bound to a specific deployed contract.
func NewDelegatedManagerSet(address common.Address, backend bind.ContractBackend) (*DelegatedManagerSet, error) {
	contract, err := bindDelegatedManagerSet(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelegatedManagerSet{DelegatedManagerSetCaller: DelegatedManagerSetCaller{contract: contract}, DelegatedManagerSetTransactor: DelegatedManagerSetTransactor{contract: contract}, DelegatedManagerSetFilterer: DelegatedManagerSetFilterer{contract: contract}}, nil
}

// NewDelegatedManagerSetCaller creates a new read-only instance of DelegatedManagerSet, bound to a specific deployed contract.
func NewDelegatedManagerSetCaller(address common.Address, caller bind.ContractCaller) (*DelegatedManagerSetCaller, error) {
	contract, err := bindDelegatedManagerSet(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegatedManagerSetCaller{contract: contract}, nil
}

// NewDelegatedManagerSetTransactor creates a new write-only instance of DelegatedManagerSet, bound to a specific deployed contract.
func NewDelegatedManagerSetTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegatedManagerSetTransactor, error) {
	contract, err := bindDelegatedManagerSet(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegatedManagerSetTransactor{contract: contract}, nil
}

// NewDelegatedManagerSetFilterer creates a new log filterer instance of DelegatedManagerSet, bound to a specific deployed contract.
func NewDelegatedManagerSetFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegatedManagerSetFilterer, error) {
	contract, err := bindDelegatedManagerSet(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegatedManagerSetFilterer{contract: contract}, nil
}

// bindDelegatedManagerSet binds a generic wrapper to an already deployed contract.
func bindDelegatedManagerSet(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DelegatedManagerSetMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegatedManagerSet *DelegatedManagerSetRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegatedManagerSet.Contract.DelegatedManagerSetCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegatedManagerSet *DelegatedManagerSetRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.DelegatedManagerSetTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegatedManagerSet *DelegatedManagerSetRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.DelegatedManagerSetTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegatedManagerSet *DelegatedManagerSetCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegatedManagerSet.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegatedManagerSet *DelegatedManagerSetTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegatedManagerSet *DelegatedManagerSetTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.contract.Transact(opts, method, params...)
}

// GetManagerSet is a free data retrieval call binding the contract method 0xa7b6250e.
//
// Solidity: function getManagerSet(uint16 chainId, uint32 index) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) GetManagerSet(opts *bind.CallOpts, chainId uint16, index uint32) ([]byte, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "getManagerSet", chainId, index)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetManagerSet is a free data retrieval call binding the contract method 0xa7b6250e.
//
// Solidity: function getManagerSet(uint16 chainId, uint32 index) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetSession) GetManagerSet(chainId uint16, index uint32) ([]byte, error) {
	return _DelegatedManagerSet.Contract.GetManagerSet(&_DelegatedManagerSet.CallOpts, chainId, index)
}

// GetManagerSet is a free data retrieval call binding the contract method 0xa7b6250e.
//
// Solidity: function getManagerSet(uint16 chainId, uint32 index) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) GetManagerSet(chainId uint16, index uint32) ([]byte, error) {
	return _DelegatedManagerSet.Contract.GetManagerSet(&_DelegatedManagerSet.CallOpts, chainId, index)
}
