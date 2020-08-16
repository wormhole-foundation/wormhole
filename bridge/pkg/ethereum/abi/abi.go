// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.
//
// TODO(leo): document how to regenerate

package abi

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

// WormholeGuardianSet is an auto generated low-level Go binding around an user-defined struct.
type WormholeGuardianSet struct {
	X              *big.Int
	Parity         uint8
	ExpirationTime uint32
}

// WormholeBridgeABI is the input ABI used to generate the binding from.
const WormholeBridgeABI = "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"parity\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"initial_guardian_set\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"wrapped_asset_master\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"parity\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"indexed\":true,\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"oldGuardian\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"parity\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"indexed\":true,\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"newGuardian\",\"type\":\"tuple\"}],\"name\":\"LogGuardianSetChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"token_chain\",\"type\":\"uint8\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"token\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sender\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"LogTokensLocked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sender\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"LogTokensUnlocked\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"guardian_set_index\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"}],\"name\":\"lockAssets\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"}],\"name\":\"lockETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vaa\",\"type\":\"bytes\"}],\"name\":\"submitVAA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vaa_expiry\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wrappedAssetMaster\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]"

// WormholeBridge is an auto generated Go binding around an Ethereum contract.
type WormholeBridge struct {
	WormholeBridgeCaller     // Read-only binding to the contract
	WormholeBridgeTransactor // Write-only binding to the contract
	WormholeBridgeFilterer   // Log filterer for contract events
}

// WormholeBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type WormholeBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WormholeBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type WormholeBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WormholeBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type WormholeBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WormholeBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type WormholeBridgeSession struct {
	Contract     *WormholeBridge   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// WormholeBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type WormholeBridgeCallerSession struct {
	Contract *WormholeBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// WormholeBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type WormholeBridgeTransactorSession struct {
	Contract     *WormholeBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// WormholeBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type WormholeBridgeRaw struct {
	Contract *WormholeBridge // Generic contract binding to access the raw methods on
}

// WormholeBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type WormholeBridgeCallerRaw struct {
	Contract *WormholeBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// WormholeBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type WormholeBridgeTransactorRaw struct {
	Contract *WormholeBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewWormholeBridge creates a new instance of WormholeBridge, bound to a specific deployed contract.
func NewWormholeBridge(address common.Address, backend bind.ContractBackend) (*WormholeBridge, error) {
	contract, err := bindWormholeBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &WormholeBridge{WormholeBridgeCaller: WormholeBridgeCaller{contract: contract}, WormholeBridgeTransactor: WormholeBridgeTransactor{contract: contract}, WormholeBridgeFilterer: WormholeBridgeFilterer{contract: contract}}, nil
}

// NewWormholeBridgeCaller creates a new read-only instance of WormholeBridge, bound to a specific deployed contract.
func NewWormholeBridgeCaller(address common.Address, caller bind.ContractCaller) (*WormholeBridgeCaller, error) {
	contract, err := bindWormholeBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeCaller{contract: contract}, nil
}

// NewWormholeBridgeTransactor creates a new write-only instance of WormholeBridge, bound to a specific deployed contract.
func NewWormholeBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*WormholeBridgeTransactor, error) {
	contract, err := bindWormholeBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeTransactor{contract: contract}, nil
}

// NewWormholeBridgeFilterer creates a new log filterer instance of WormholeBridge, bound to a specific deployed contract.
func NewWormholeBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*WormholeBridgeFilterer, error) {
	contract, err := bindWormholeBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeFilterer{contract: contract}, nil
}

// bindWormholeBridge binds a generic wrapper to an already deployed contract.
func bindWormholeBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(WormholeBridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WormholeBridge *WormholeBridgeRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _WormholeBridge.Contract.WormholeBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WormholeBridge *WormholeBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WormholeBridge.Contract.WormholeBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WormholeBridge *WormholeBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WormholeBridge.Contract.WormholeBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WormholeBridge *WormholeBridgeCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _WormholeBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WormholeBridge *WormholeBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WormholeBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WormholeBridge *WormholeBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WormholeBridge.Contract.contract.Transact(opts, method, params...)
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_WormholeBridge *WormholeBridgeCaller) GuardianSetIndex(opts *bind.CallOpts) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _WormholeBridge.contract.Call(opts, out, "guardian_set_index")
	return *ret0, err
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_WormholeBridge *WormholeBridgeSession) GuardianSetIndex() (uint32, error) {
	return _WormholeBridge.Contract.GuardianSetIndex(&_WormholeBridge.CallOpts)
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_WormholeBridge *WormholeBridgeCallerSession) GuardianSetIndex() (uint32, error) {
	return _WormholeBridge.Contract.GuardianSetIndex(&_WormholeBridge.CallOpts)
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_WormholeBridge *WormholeBridgeCaller) VaaExpiry(opts *bind.CallOpts) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _WormholeBridge.contract.Call(opts, out, "vaa_expiry")
	return *ret0, err
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_WormholeBridge *WormholeBridgeSession) VaaExpiry() (uint32, error) {
	return _WormholeBridge.Contract.VaaExpiry(&_WormholeBridge.CallOpts)
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_WormholeBridge *WormholeBridgeCallerSession) VaaExpiry() (uint32, error) {
	return _WormholeBridge.Contract.VaaExpiry(&_WormholeBridge.CallOpts)
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_WormholeBridge *WormholeBridgeCaller) WrappedAssetMaster(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _WormholeBridge.contract.Call(opts, out, "wrappedAssetMaster")
	return *ret0, err
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_WormholeBridge *WormholeBridgeSession) WrappedAssetMaster() (common.Address, error) {
	return _WormholeBridge.Contract.WrappedAssetMaster(&_WormholeBridge.CallOpts)
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_WormholeBridge *WormholeBridgeCallerSession) WrappedAssetMaster() (common.Address, error) {
	return _WormholeBridge.Contract.WrappedAssetMaster(&_WormholeBridge.CallOpts)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_WormholeBridge *WormholeBridgeTransactor) LockAssets(opts *bind.TransactOpts, asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.contract.Transact(opts, "lockAssets", asset, amount, recipient, target_chain)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_WormholeBridge *WormholeBridgeSession) LockAssets(asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.Contract.LockAssets(&_WormholeBridge.TransactOpts, asset, amount, recipient, target_chain)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_WormholeBridge *WormholeBridgeTransactorSession) LockAssets(asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.Contract.LockAssets(&_WormholeBridge.TransactOpts, asset, amount, recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_WormholeBridge *WormholeBridgeTransactor) LockETH(opts *bind.TransactOpts, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.contract.Transact(opts, "lockETH", recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_WormholeBridge *WormholeBridgeSession) LockETH(recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.Contract.LockETH(&_WormholeBridge.TransactOpts, recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_WormholeBridge *WormholeBridgeTransactorSession) LockETH(recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _WormholeBridge.Contract.LockETH(&_WormholeBridge.TransactOpts, recipient, target_chain)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_WormholeBridge *WormholeBridgeTransactor) SubmitVAA(opts *bind.TransactOpts, vaa []byte) (*types.Transaction, error) {
	return _WormholeBridge.contract.Transact(opts, "submitVAA", vaa)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_WormholeBridge *WormholeBridgeSession) SubmitVAA(vaa []byte) (*types.Transaction, error) {
	return _WormholeBridge.Contract.SubmitVAA(&_WormholeBridge.TransactOpts, vaa)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_WormholeBridge *WormholeBridgeTransactorSession) SubmitVAA(vaa []byte) (*types.Transaction, error) {
	return _WormholeBridge.Contract.SubmitVAA(&_WormholeBridge.TransactOpts, vaa)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_WormholeBridge *WormholeBridgeTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _WormholeBridge.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_WormholeBridge *WormholeBridgeSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _WormholeBridge.Contract.Fallback(&_WormholeBridge.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_WormholeBridge *WormholeBridgeTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _WormholeBridge.Contract.Fallback(&_WormholeBridge.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_WormholeBridge *WormholeBridgeTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WormholeBridge.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_WormholeBridge *WormholeBridgeSession) Receive() (*types.Transaction, error) {
	return _WormholeBridge.Contract.Receive(&_WormholeBridge.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_WormholeBridge *WormholeBridgeTransactorSession) Receive() (*types.Transaction, error) {
	return _WormholeBridge.Contract.Receive(&_WormholeBridge.TransactOpts)
}

// WormholeBridgeLogGuardianSetChangedIterator is returned from FilterLogGuardianSetChanged and is used to iterate over the raw logs and unpacked data for LogGuardianSetChanged events raised by the WormholeBridge contract.
type WormholeBridgeLogGuardianSetChangedIterator struct {
	Event *WormholeBridgeLogGuardianSetChanged // Event containing the contract specifics and raw log

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
func (it *WormholeBridgeLogGuardianSetChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WormholeBridgeLogGuardianSetChanged)
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
		it.Event = new(WormholeBridgeLogGuardianSetChanged)
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
func (it *WormholeBridgeLogGuardianSetChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WormholeBridgeLogGuardianSetChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WormholeBridgeLogGuardianSetChanged represents a LogGuardianSetChanged event raised by the WormholeBridge contract.
type WormholeBridgeLogGuardianSetChanged struct {
	OldGuardian WormholeGuardianSet
	NewGuardian WormholeGuardianSet
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterLogGuardianSetChanged is a free log retrieval operation binding the contract event 0x776a7721d091beb15fb219d7be3c92b83fa7c10428af15a7312461bc3bc52e0b.
//
// Solidity: event LogGuardianSetChanged((uint256,uint8,uint32) indexed oldGuardian, (uint256,uint8,uint32) indexed newGuardian)
func (_WormholeBridge *WormholeBridgeFilterer) FilterLogGuardianSetChanged(opts *bind.FilterOpts, oldGuardian []WormholeGuardianSet, newGuardian []WormholeGuardianSet) (*WormholeBridgeLogGuardianSetChangedIterator, error) {

	var oldGuardianRule []interface{}
	for _, oldGuardianItem := range oldGuardian {
		oldGuardianRule = append(oldGuardianRule, oldGuardianItem)
	}
	var newGuardianRule []interface{}
	for _, newGuardianItem := range newGuardian {
		newGuardianRule = append(newGuardianRule, newGuardianItem)
	}

	logs, sub, err := _WormholeBridge.contract.FilterLogs(opts, "LogGuardianSetChanged", oldGuardianRule, newGuardianRule)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeLogGuardianSetChangedIterator{contract: _WormholeBridge.contract, event: "LogGuardianSetChanged", logs: logs, sub: sub}, nil
}

// WatchLogGuardianSetChanged is a free log subscription operation binding the contract event 0x776a7721d091beb15fb219d7be3c92b83fa7c10428af15a7312461bc3bc52e0b.
//
// Solidity: event LogGuardianSetChanged((uint256,uint8,uint32) indexed oldGuardian, (uint256,uint8,uint32) indexed newGuardian)
func (_WormholeBridge *WormholeBridgeFilterer) WatchLogGuardianSetChanged(opts *bind.WatchOpts, sink chan<- *WormholeBridgeLogGuardianSetChanged, oldGuardian []WormholeGuardianSet, newGuardian []WormholeGuardianSet) (event.Subscription, error) {

	var oldGuardianRule []interface{}
	for _, oldGuardianItem := range oldGuardian {
		oldGuardianRule = append(oldGuardianRule, oldGuardianItem)
	}
	var newGuardianRule []interface{}
	for _, newGuardianItem := range newGuardian {
		newGuardianRule = append(newGuardianRule, newGuardianItem)
	}

	logs, sub, err := _WormholeBridge.contract.WatchLogs(opts, "LogGuardianSetChanged", oldGuardianRule, newGuardianRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WormholeBridgeLogGuardianSetChanged)
				if err := _WormholeBridge.contract.UnpackLog(event, "LogGuardianSetChanged", log); err != nil {
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

// ParseLogGuardianSetChanged is a log parse operation binding the contract event 0x776a7721d091beb15fb219d7be3c92b83fa7c10428af15a7312461bc3bc52e0b.
//
// Solidity: event LogGuardianSetChanged((uint256,uint8,uint32) indexed oldGuardian, (uint256,uint8,uint32) indexed newGuardian)
func (_WormholeBridge *WormholeBridgeFilterer) ParseLogGuardianSetChanged(log types.Log) (*WormholeBridgeLogGuardianSetChanged, error) {
	event := new(WormholeBridgeLogGuardianSetChanged)
	if err := _WormholeBridge.contract.UnpackLog(event, "LogGuardianSetChanged", log); err != nil {
		return nil, err
	}
	return event, nil
}

// WormholeBridgeLogTokensLockedIterator is returned from FilterLogTokensLocked and is used to iterate over the raw logs and unpacked data for LogTokensLocked events raised by the WormholeBridge contract.
type WormholeBridgeLogTokensLockedIterator struct {
	Event *WormholeBridgeLogTokensLocked // Event containing the contract specifics and raw log

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
func (it *WormholeBridgeLogTokensLockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WormholeBridgeLogTokensLocked)
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
		it.Event = new(WormholeBridgeLogTokensLocked)
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
func (it *WormholeBridgeLogTokensLockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WormholeBridgeLogTokensLockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WormholeBridgeLogTokensLocked represents a LogTokensLocked event raised by the WormholeBridge contract.
type WormholeBridgeLogTokensLocked struct {
	TargetChain uint8
	TokenChain  uint8
	Token       [32]byte
	Sender      [32]byte
	Recipient   [32]byte
	Amount      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterLogTokensLocked is a free log retrieval operation binding the contract event 0x84b445260a99044cc9529b3033663c078031a14e31f3c255ff02c62667bab14b.
//
// Solidity: event LogTokensLocked(uint8 target_chain, uint8 token_chain, bytes32 indexed token, bytes32 indexed sender, bytes32 recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) FilterLogTokensLocked(opts *bind.FilterOpts, token [][32]byte, sender [][32]byte) (*WormholeBridgeLogTokensLockedIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _WormholeBridge.contract.FilterLogs(opts, "LogTokensLocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeLogTokensLockedIterator{contract: _WormholeBridge.contract, event: "LogTokensLocked", logs: logs, sub: sub}, nil
}

// WatchLogTokensLocked is a free log subscription operation binding the contract event 0x84b445260a99044cc9529b3033663c078031a14e31f3c255ff02c62667bab14b.
//
// Solidity: event LogTokensLocked(uint8 target_chain, uint8 token_chain, bytes32 indexed token, bytes32 indexed sender, bytes32 recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) WatchLogTokensLocked(opts *bind.WatchOpts, sink chan<- *WormholeBridgeLogTokensLocked, token [][32]byte, sender [][32]byte) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _WormholeBridge.contract.WatchLogs(opts, "LogTokensLocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WormholeBridgeLogTokensLocked)
				if err := _WormholeBridge.contract.UnpackLog(event, "LogTokensLocked", log); err != nil {
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

// ParseLogTokensLocked is a log parse operation binding the contract event 0x84b445260a99044cc9529b3033663c078031a14e31f3c255ff02c62667bab14b.
//
// Solidity: event LogTokensLocked(uint8 target_chain, uint8 token_chain, bytes32 indexed token, bytes32 indexed sender, bytes32 recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) ParseLogTokensLocked(log types.Log) (*WormholeBridgeLogTokensLocked, error) {
	event := new(WormholeBridgeLogTokensLocked)
	if err := _WormholeBridge.contract.UnpackLog(event, "LogTokensLocked", log); err != nil {
		return nil, err
	}
	return event, nil
}

// WormholeBridgeLogTokensUnlockedIterator is returned from FilterLogTokensUnlocked and is used to iterate over the raw logs and unpacked data for LogTokensUnlocked events raised by the WormholeBridge contract.
type WormholeBridgeLogTokensUnlockedIterator struct {
	Event *WormholeBridgeLogTokensUnlocked // Event containing the contract specifics and raw log

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
func (it *WormholeBridgeLogTokensUnlockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WormholeBridgeLogTokensUnlocked)
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
		it.Event = new(WormholeBridgeLogTokensUnlocked)
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
func (it *WormholeBridgeLogTokensUnlockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WormholeBridgeLogTokensUnlockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WormholeBridgeLogTokensUnlocked represents a LogTokensUnlocked event raised by the WormholeBridge contract.
type WormholeBridgeLogTokensUnlocked struct {
	Token     common.Address
	Sender    [32]byte
	Recipient common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterLogTokensUnlocked is a free log retrieval operation binding the contract event 0x762f40dceecda77485ebb6a807ba1ba35b861c5d34fe9bffbb94826a0dc0f101.
//
// Solidity: event LogTokensUnlocked(address indexed token, bytes32 indexed sender, address recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) FilterLogTokensUnlocked(opts *bind.FilterOpts, token []common.Address, sender [][32]byte) (*WormholeBridgeLogTokensUnlockedIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _WormholeBridge.contract.FilterLogs(opts, "LogTokensUnlocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &WormholeBridgeLogTokensUnlockedIterator{contract: _WormholeBridge.contract, event: "LogTokensUnlocked", logs: logs, sub: sub}, nil
}

// WatchLogTokensUnlocked is a free log subscription operation binding the contract event 0x762f40dceecda77485ebb6a807ba1ba35b861c5d34fe9bffbb94826a0dc0f101.
//
// Solidity: event LogTokensUnlocked(address indexed token, bytes32 indexed sender, address recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) WatchLogTokensUnlocked(opts *bind.WatchOpts, sink chan<- *WormholeBridgeLogTokensUnlocked, token []common.Address, sender [][32]byte) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _WormholeBridge.contract.WatchLogs(opts, "LogTokensUnlocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WormholeBridgeLogTokensUnlocked)
				if err := _WormholeBridge.contract.UnpackLog(event, "LogTokensUnlocked", log); err != nil {
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

// ParseLogTokensUnlocked is a log parse operation binding the contract event 0x762f40dceecda77485ebb6a807ba1ba35b861c5d34fe9bffbb94826a0dc0f101.
//
// Solidity: event LogTokensUnlocked(address indexed token, bytes32 indexed sender, address recipient, uint256 amount)
func (_WormholeBridge *WormholeBridgeFilterer) ParseLogTokensUnlocked(log types.Log) (*WormholeBridgeLogTokensUnlocked, error) {
	event := new(WormholeBridgeLogTokensUnlocked)
	if err := _WormholeBridge.contract.UnpackLog(event, "LogTokensUnlocked", log); err != nil {
		return nil, err
	}
	return event, nil
}
