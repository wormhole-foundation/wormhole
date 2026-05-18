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

// IDelegatedManagerSetManagerSetUpdate is an auto generated low-level Go binding around an user-defined struct.
type IDelegatedManagerSetManagerSetUpdate struct {
	Module          [32]byte
	Action          uint8
	ChainId         uint16
	ManagerChainId  uint16
	ManagerSetIndex uint32
	ManagerSet      []byte
}

// DelegatedManagerSetMetaData contains all meta data concerning the DelegatedManagerSet contract.
var DelegatedManagerSetMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_wormhole\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WORMHOLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIWormhole\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"consumedGovernanceActions\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCurrentManagerSet\",\"inputs\":[{\"name\":\"chainId\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCurrentManagerSetIndex\",\"inputs\":[{\"name\":\"chainId\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getManagerSet\",\"inputs\":[{\"name\":\"chainId\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"index\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"parseManagerSetUpdate\",\"inputs\":[{\"name\":\"encoded\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"update\",\"type\":\"tuple\",\"internalType\":\"structIDelegatedManagerSet.ManagerSetUpdate\",\"components\":[{\"name\":\"module\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"action\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"chainId\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"managerChainId\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"managerSetIndex\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"managerSet\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"submitNewManagerSet\",\"inputs\":[{\"name\":\"encodedVm\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"NewManagerSet\",\"inputs\":[{\"name\":\"chain\",\"type\":\"uint16\",\"indexed\":false,\"internalType\":\"uint16\"},{\"name\":\"index\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AlreadyConsumed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidAction\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidChain\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGovernanceChain\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGovernanceContract\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidGuardianSet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidIndex\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidModule\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidVAA\",\"inputs\":[{\"name\":\"reason\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
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

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) VERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_DelegatedManagerSet *DelegatedManagerSetSession) VERSION() (string, error) {
	return _DelegatedManagerSet.Contract.VERSION(&_DelegatedManagerSet.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) VERSION() (string, error) {
	return _DelegatedManagerSet.Contract.VERSION(&_DelegatedManagerSet.CallOpts)
}

// WORMHOLE is a free data retrieval call binding the contract method 0x35e78cfe.
//
// Solidity: function WORMHOLE() view returns(address)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) WORMHOLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "WORMHOLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// WORMHOLE is a free data retrieval call binding the contract method 0x35e78cfe.
//
// Solidity: function WORMHOLE() view returns(address)
func (_DelegatedManagerSet *DelegatedManagerSetSession) WORMHOLE() (common.Address, error) {
	return _DelegatedManagerSet.Contract.WORMHOLE(&_DelegatedManagerSet.CallOpts)
}

// WORMHOLE is a free data retrieval call binding the contract method 0x35e78cfe.
//
// Solidity: function WORMHOLE() view returns(address)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) WORMHOLE() (common.Address, error) {
	return _DelegatedManagerSet.Contract.WORMHOLE(&_DelegatedManagerSet.CallOpts)
}

// ConsumedGovernanceActions is a free data retrieval call binding the contract method 0x5b4af028.
//
// Solidity: function consumedGovernanceActions(bytes32 ) view returns(bool)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) ConsumedGovernanceActions(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "consumedGovernanceActions", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ConsumedGovernanceActions is a free data retrieval call binding the contract method 0x5b4af028.
//
// Solidity: function consumedGovernanceActions(bytes32 ) view returns(bool)
func (_DelegatedManagerSet *DelegatedManagerSetSession) ConsumedGovernanceActions(arg0 [32]byte) (bool, error) {
	return _DelegatedManagerSet.Contract.ConsumedGovernanceActions(&_DelegatedManagerSet.CallOpts, arg0)
}

// ConsumedGovernanceActions is a free data retrieval call binding the contract method 0x5b4af028.
//
// Solidity: function consumedGovernanceActions(bytes32 ) view returns(bool)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) ConsumedGovernanceActions(arg0 [32]byte) (bool, error) {
	return _DelegatedManagerSet.Contract.ConsumedGovernanceActions(&_DelegatedManagerSet.CallOpts, arg0)
}

// GetCurrentManagerSet is a free data retrieval call binding the contract method 0xf81c174b.
//
// Solidity: function getCurrentManagerSet(uint16 chainId) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) GetCurrentManagerSet(opts *bind.CallOpts, chainId uint16) ([]byte, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "getCurrentManagerSet", chainId)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetCurrentManagerSet is a free data retrieval call binding the contract method 0xf81c174b.
//
// Solidity: function getCurrentManagerSet(uint16 chainId) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetSession) GetCurrentManagerSet(chainId uint16) ([]byte, error) {
	return _DelegatedManagerSet.Contract.GetCurrentManagerSet(&_DelegatedManagerSet.CallOpts, chainId)
}

// GetCurrentManagerSet is a free data retrieval call binding the contract method 0xf81c174b.
//
// Solidity: function getCurrentManagerSet(uint16 chainId) view returns(bytes)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) GetCurrentManagerSet(chainId uint16) ([]byte, error) {
	return _DelegatedManagerSet.Contract.GetCurrentManagerSet(&_DelegatedManagerSet.CallOpts, chainId)
}

// GetCurrentManagerSetIndex is a free data retrieval call binding the contract method 0xca779a52.
//
// Solidity: function getCurrentManagerSetIndex(uint16 chainId) view returns(uint32)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) GetCurrentManagerSetIndex(opts *bind.CallOpts, chainId uint16) (uint32, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "getCurrentManagerSetIndex", chainId)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// GetCurrentManagerSetIndex is a free data retrieval call binding the contract method 0xca779a52.
//
// Solidity: function getCurrentManagerSetIndex(uint16 chainId) view returns(uint32)
func (_DelegatedManagerSet *DelegatedManagerSetSession) GetCurrentManagerSetIndex(chainId uint16) (uint32, error) {
	return _DelegatedManagerSet.Contract.GetCurrentManagerSetIndex(&_DelegatedManagerSet.CallOpts, chainId)
}

// GetCurrentManagerSetIndex is a free data retrieval call binding the contract method 0xca779a52.
//
// Solidity: function getCurrentManagerSetIndex(uint16 chainId) view returns(uint32)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) GetCurrentManagerSetIndex(chainId uint16) (uint32, error) {
	return _DelegatedManagerSet.Contract.GetCurrentManagerSetIndex(&_DelegatedManagerSet.CallOpts, chainId)
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

// ParseManagerSetUpdate is a free data retrieval call binding the contract method 0xa8dd14a2.
//
// Solidity: function parseManagerSetUpdate(bytes encoded) pure returns((bytes32,uint8,uint16,uint16,uint32,bytes) update)
func (_DelegatedManagerSet *DelegatedManagerSetCaller) ParseManagerSetUpdate(opts *bind.CallOpts, encoded []byte) (IDelegatedManagerSetManagerSetUpdate, error) {
	var out []interface{}
	err := _DelegatedManagerSet.contract.Call(opts, &out, "parseManagerSetUpdate", encoded)

	if err != nil {
		return *new(IDelegatedManagerSetManagerSetUpdate), err
	}

	out0 := *abi.ConvertType(out[0], new(IDelegatedManagerSetManagerSetUpdate)).(*IDelegatedManagerSetManagerSetUpdate)

	return out0, err

}

// ParseManagerSetUpdate is a free data retrieval call binding the contract method 0xa8dd14a2.
//
// Solidity: function parseManagerSetUpdate(bytes encoded) pure returns((bytes32,uint8,uint16,uint16,uint32,bytes) update)
func (_DelegatedManagerSet *DelegatedManagerSetSession) ParseManagerSetUpdate(encoded []byte) (IDelegatedManagerSetManagerSetUpdate, error) {
	return _DelegatedManagerSet.Contract.ParseManagerSetUpdate(&_DelegatedManagerSet.CallOpts, encoded)
}

// ParseManagerSetUpdate is a free data retrieval call binding the contract method 0xa8dd14a2.
//
// Solidity: function parseManagerSetUpdate(bytes encoded) pure returns((bytes32,uint8,uint16,uint16,uint32,bytes) update)
func (_DelegatedManagerSet *DelegatedManagerSetCallerSession) ParseManagerSetUpdate(encoded []byte) (IDelegatedManagerSetManagerSetUpdate, error) {
	return _DelegatedManagerSet.Contract.ParseManagerSetUpdate(&_DelegatedManagerSet.CallOpts, encoded)
}

// SubmitNewManagerSet is a paid mutator transaction binding the contract method 0x5f8d96c9.
//
// Solidity: function submitNewManagerSet(bytes encodedVm) returns()
func (_DelegatedManagerSet *DelegatedManagerSetTransactor) SubmitNewManagerSet(opts *bind.TransactOpts, encodedVm []byte) (*types.Transaction, error) {
	return _DelegatedManagerSet.contract.Transact(opts, "submitNewManagerSet", encodedVm)
}

// SubmitNewManagerSet is a paid mutator transaction binding the contract method 0x5f8d96c9.
//
// Solidity: function submitNewManagerSet(bytes encodedVm) returns()
func (_DelegatedManagerSet *DelegatedManagerSetSession) SubmitNewManagerSet(encodedVm []byte) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.SubmitNewManagerSet(&_DelegatedManagerSet.TransactOpts, encodedVm)
}

// SubmitNewManagerSet is a paid mutator transaction binding the contract method 0x5f8d96c9.
//
// Solidity: function submitNewManagerSet(bytes encodedVm) returns()
func (_DelegatedManagerSet *DelegatedManagerSetTransactorSession) SubmitNewManagerSet(encodedVm []byte) (*types.Transaction, error) {
	return _DelegatedManagerSet.Contract.SubmitNewManagerSet(&_DelegatedManagerSet.TransactOpts, encodedVm)
}

// DelegatedManagerSetNewManagerSetIterator is returned from FilterNewManagerSet and is used to iterate over the raw logs and unpacked data for NewManagerSet events raised by the DelegatedManagerSet contract.
type DelegatedManagerSetNewManagerSetIterator struct {
	Event *DelegatedManagerSetNewManagerSet // Event containing the contract specifics and raw log

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
func (it *DelegatedManagerSetNewManagerSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegatedManagerSetNewManagerSet)
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
		it.Event = new(DelegatedManagerSetNewManagerSet)
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
func (it *DelegatedManagerSetNewManagerSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegatedManagerSetNewManagerSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegatedManagerSetNewManagerSet represents a NewManagerSet event raised by the DelegatedManagerSet contract.
type DelegatedManagerSetNewManagerSet struct {
	Chain uint16
	Index uint32
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNewManagerSet is a free log retrieval operation binding the contract event 0x923bb3e008cbe2e6e13010f8fc996a0c8c62d29f8cf1b252f663d8f396843b9d.
//
// Solidity: event NewManagerSet(uint16 chain, uint32 index)
func (_DelegatedManagerSet *DelegatedManagerSetFilterer) FilterNewManagerSet(opts *bind.FilterOpts) (*DelegatedManagerSetNewManagerSetIterator, error) {

	logs, sub, err := _DelegatedManagerSet.contract.FilterLogs(opts, "NewManagerSet")
	if err != nil {
		return nil, err
	}
	return &DelegatedManagerSetNewManagerSetIterator{contract: _DelegatedManagerSet.contract, event: "NewManagerSet", logs: logs, sub: sub}, nil
}

// WatchNewManagerSet is a free log subscription operation binding the contract event 0x923bb3e008cbe2e6e13010f8fc996a0c8c62d29f8cf1b252f663d8f396843b9d.
//
// Solidity: event NewManagerSet(uint16 chain, uint32 index)
func (_DelegatedManagerSet *DelegatedManagerSetFilterer) WatchNewManagerSet(opts *bind.WatchOpts, sink chan<- *DelegatedManagerSetNewManagerSet) (event.Subscription, error) {

	logs, sub, err := _DelegatedManagerSet.contract.WatchLogs(opts, "NewManagerSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegatedManagerSetNewManagerSet)
				if err := _DelegatedManagerSet.contract.UnpackLog(event, "NewManagerSet", log); err != nil {
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

// ParseNewManagerSet is a log parse operation binding the contract event 0x923bb3e008cbe2e6e13010f8fc996a0c8c62d29f8cf1b252f663d8f396843b9d.
//
// Solidity: event NewManagerSet(uint16 chain, uint32 index)
func (_DelegatedManagerSet *DelegatedManagerSetFilterer) ParseNewManagerSet(log types.Log) (*DelegatedManagerSetNewManagerSet, error) {
	event := new(DelegatedManagerSetNewManagerSet)
	if err := _DelegatedManagerSet.contract.UnpackLog(event, "NewManagerSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
