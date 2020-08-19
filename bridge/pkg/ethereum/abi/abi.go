// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

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
	Keys           []common.Address
	ExpirationTime uint32
}

// AbiABI is the input ABI used to generate the binding from.
const AbiABI = "[{\"inputs\":[{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"initial_guardian_set\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"wrapped_asset_master\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_vaa_expiry\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"oldGuardianIndex\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"newGuardianIndex\",\"type\":\"uint32\"}],\"name\":\"LogGuardianSetChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"token_chain\",\"type\":\"uint8\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"token\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sender\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"LogTokensLocked\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"guardian_set_index\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"name\":\"guardian_sets\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isWrappedAsset\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vaa_expiry\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wrappedAssetMaster\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"wrappedAssets\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"getGuardianSet\",\"outputs\":[{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"gs\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vaa\",\"type\":\"bytes\"}],\"name\":\"submitVAA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"}],\"name\":\"lockAssets\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"}],\"name\":\"lockETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]"

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
func (_Abi *AbiRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
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
func (_Abi *AbiCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
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

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 idx) view returns((address[],uint32) gs)
func (_Abi *AbiCaller) GetGuardianSet(opts *bind.CallOpts, idx uint32) (WormholeGuardianSet, error) {
	var (
		ret0 = new(WormholeGuardianSet)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "getGuardianSet", idx)
	return *ret0, err
}

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 idx) view returns((address[],uint32) gs)
func (_Abi *AbiSession) GetGuardianSet(idx uint32) (WormholeGuardianSet, error) {
	return _Abi.Contract.GetGuardianSet(&_Abi.CallOpts, idx)
}

// GetGuardianSet is a free data retrieval call binding the contract method 0xf951975a.
//
// Solidity: function getGuardianSet(uint32 idx) view returns((address[],uint32) gs)
func (_Abi *AbiCallerSession) GetGuardianSet(idx uint32) (WormholeGuardianSet, error) {
	return _Abi.Contract.GetGuardianSet(&_Abi.CallOpts, idx)
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_Abi *AbiCaller) GuardianSetIndex(opts *bind.CallOpts) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "guardian_set_index")
	return *ret0, err
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_Abi *AbiSession) GuardianSetIndex() (uint32, error) {
	return _Abi.Contract.GuardianSetIndex(&_Abi.CallOpts)
}

// GuardianSetIndex is a free data retrieval call binding the contract method 0x822d82b3.
//
// Solidity: function guardian_set_index() view returns(uint32)
func (_Abi *AbiCallerSession) GuardianSetIndex() (uint32, error) {
	return _Abi.Contract.GuardianSetIndex(&_Abi.CallOpts)
}

// GuardianSets is a free data retrieval call binding the contract method 0x42b0aefa.
//
// Solidity: function guardian_sets(uint32 ) view returns(uint32 expiration_time)
func (_Abi *AbiCaller) GuardianSets(opts *bind.CallOpts, arg0 uint32) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "guardian_sets", arg0)
	return *ret0, err
}

// GuardianSets is a free data retrieval call binding the contract method 0x42b0aefa.
//
// Solidity: function guardian_sets(uint32 ) view returns(uint32 expiration_time)
func (_Abi *AbiSession) GuardianSets(arg0 uint32) (uint32, error) {
	return _Abi.Contract.GuardianSets(&_Abi.CallOpts, arg0)
}

// GuardianSets is a free data retrieval call binding the contract method 0x42b0aefa.
//
// Solidity: function guardian_sets(uint32 ) view returns(uint32 expiration_time)
func (_Abi *AbiCallerSession) GuardianSets(arg0 uint32) (uint32, error) {
	return _Abi.Contract.GuardianSets(&_Abi.CallOpts, arg0)
}

// IsWrappedAsset is a free data retrieval call binding the contract method 0x1a2be4da.
//
// Solidity: function isWrappedAsset(address ) view returns(bool)
func (_Abi *AbiCaller) IsWrappedAsset(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "isWrappedAsset", arg0)
	return *ret0, err
}

// IsWrappedAsset is a free data retrieval call binding the contract method 0x1a2be4da.
//
// Solidity: function isWrappedAsset(address ) view returns(bool)
func (_Abi *AbiSession) IsWrappedAsset(arg0 common.Address) (bool, error) {
	return _Abi.Contract.IsWrappedAsset(&_Abi.CallOpts, arg0)
}

// IsWrappedAsset is a free data retrieval call binding the contract method 0x1a2be4da.
//
// Solidity: function isWrappedAsset(address ) view returns(bool)
func (_Abi *AbiCallerSession) IsWrappedAsset(arg0 common.Address) (bool, error) {
	return _Abi.Contract.IsWrappedAsset(&_Abi.CallOpts, arg0)
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_Abi *AbiCaller) VaaExpiry(opts *bind.CallOpts) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "vaa_expiry")
	return *ret0, err
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_Abi *AbiSession) VaaExpiry() (uint32, error) {
	return _Abi.Contract.VaaExpiry(&_Abi.CallOpts)
}

// VaaExpiry is a free data retrieval call binding the contract method 0x7f04d9e6.
//
// Solidity: function vaa_expiry() view returns(uint32)
func (_Abi *AbiCallerSession) VaaExpiry() (uint32, error) {
	return _Abi.Contract.VaaExpiry(&_Abi.CallOpts)
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_Abi *AbiCaller) WrappedAssetMaster(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "wrappedAssetMaster")
	return *ret0, err
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_Abi *AbiSession) WrappedAssetMaster() (common.Address, error) {
	return _Abi.Contract.WrappedAssetMaster(&_Abi.CallOpts)
}

// WrappedAssetMaster is a free data retrieval call binding the contract method 0x99da1d3c.
//
// Solidity: function wrappedAssetMaster() view returns(address)
func (_Abi *AbiCallerSession) WrappedAssetMaster() (common.Address, error) {
	return _Abi.Contract.WrappedAssetMaster(&_Abi.CallOpts)
}

// WrappedAssets is a free data retrieval call binding the contract method 0xb6694c2a.
//
// Solidity: function wrappedAssets(bytes32 ) view returns(address)
func (_Abi *AbiCaller) WrappedAssets(opts *bind.CallOpts, arg0 [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Abi.contract.Call(opts, out, "wrappedAssets", arg0)
	return *ret0, err
}

// WrappedAssets is a free data retrieval call binding the contract method 0xb6694c2a.
//
// Solidity: function wrappedAssets(bytes32 ) view returns(address)
func (_Abi *AbiSession) WrappedAssets(arg0 [32]byte) (common.Address, error) {
	return _Abi.Contract.WrappedAssets(&_Abi.CallOpts, arg0)
}

// WrappedAssets is a free data retrieval call binding the contract method 0xb6694c2a.
//
// Solidity: function wrappedAssets(bytes32 ) view returns(address)
func (_Abi *AbiCallerSession) WrappedAssets(arg0 [32]byte) (common.Address, error) {
	return _Abi.Contract.WrappedAssets(&_Abi.CallOpts, arg0)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_Abi *AbiTransactor) LockAssets(opts *bind.TransactOpts, asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "lockAssets", asset, amount, recipient, target_chain)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_Abi *AbiSession) LockAssets(asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.Contract.LockAssets(&_Abi.TransactOpts, asset, amount, recipient, target_chain)
}

// LockAssets is a paid mutator transaction binding the contract method 0xe66fd373.
//
// Solidity: function lockAssets(address asset, uint256 amount, bytes32 recipient, uint8 target_chain) returns()
func (_Abi *AbiTransactorSession) LockAssets(asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.Contract.LockAssets(&_Abi.TransactOpts, asset, amount, recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_Abi *AbiTransactor) LockETH(opts *bind.TransactOpts, recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "lockETH", recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_Abi *AbiSession) LockETH(recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.Contract.LockETH(&_Abi.TransactOpts, recipient, target_chain)
}

// LockETH is a paid mutator transaction binding the contract method 0x780e2183.
//
// Solidity: function lockETH(bytes32 recipient, uint8 target_chain) payable returns()
func (_Abi *AbiTransactorSession) LockETH(recipient [32]byte, target_chain uint8) (*types.Transaction, error) {
	return _Abi.Contract.LockETH(&_Abi.TransactOpts, recipient, target_chain)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_Abi *AbiTransactor) SubmitVAA(opts *bind.TransactOpts, vaa []byte) (*types.Transaction, error) {
	return _Abi.contract.Transact(opts, "submitVAA", vaa)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_Abi *AbiSession) SubmitVAA(vaa []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitVAA(&_Abi.TransactOpts, vaa)
}

// SubmitVAA is a paid mutator transaction binding the contract method 0x3bc0aee6.
//
// Solidity: function submitVAA(bytes vaa) returns()
func (_Abi *AbiTransactorSession) SubmitVAA(vaa []byte) (*types.Transaction, error) {
	return _Abi.Contract.SubmitVAA(&_Abi.TransactOpts, vaa)
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

// AbiLogGuardianSetChangedIterator is returned from FilterLogGuardianSetChanged and is used to iterate over the raw logs and unpacked data for LogGuardianSetChanged events raised by the Abi contract.
type AbiLogGuardianSetChangedIterator struct {
	Event *AbiLogGuardianSetChanged // Event containing the contract specifics and raw log

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
func (it *AbiLogGuardianSetChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiLogGuardianSetChanged)
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
		it.Event = new(AbiLogGuardianSetChanged)
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
func (it *AbiLogGuardianSetChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiLogGuardianSetChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiLogGuardianSetChanged represents a LogGuardianSetChanged event raised by the Abi contract.
type AbiLogGuardianSetChanged struct {
	OldGuardianIndex uint32
	NewGuardianIndex uint32
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterLogGuardianSetChanged is a free log retrieval operation binding the contract event 0xdfb80683934199683861bf00b64ecdf0984bbaf661bf27983dba382e99297a62.
//
// Solidity: event LogGuardianSetChanged(uint32 oldGuardianIndex, uint32 newGuardianIndex)
func (_Abi *AbiFilterer) FilterLogGuardianSetChanged(opts *bind.FilterOpts) (*AbiLogGuardianSetChangedIterator, error) {

	logs, sub, err := _Abi.contract.FilterLogs(opts, "LogGuardianSetChanged")
	if err != nil {
		return nil, err
	}
	return &AbiLogGuardianSetChangedIterator{contract: _Abi.contract, event: "LogGuardianSetChanged", logs: logs, sub: sub}, nil
}

// WatchLogGuardianSetChanged is a free log subscription operation binding the contract event 0xdfb80683934199683861bf00b64ecdf0984bbaf661bf27983dba382e99297a62.
//
// Solidity: event LogGuardianSetChanged(uint32 oldGuardianIndex, uint32 newGuardianIndex)
func (_Abi *AbiFilterer) WatchLogGuardianSetChanged(opts *bind.WatchOpts, sink chan<- *AbiLogGuardianSetChanged) (event.Subscription, error) {

	logs, sub, err := _Abi.contract.WatchLogs(opts, "LogGuardianSetChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiLogGuardianSetChanged)
				if err := _Abi.contract.UnpackLog(event, "LogGuardianSetChanged", log); err != nil {
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

// ParseLogGuardianSetChanged is a log parse operation binding the contract event 0xdfb80683934199683861bf00b64ecdf0984bbaf661bf27983dba382e99297a62.
//
// Solidity: event LogGuardianSetChanged(uint32 oldGuardianIndex, uint32 newGuardianIndex)
func (_Abi *AbiFilterer) ParseLogGuardianSetChanged(log types.Log) (*AbiLogGuardianSetChanged, error) {
	event := new(AbiLogGuardianSetChanged)
	if err := _Abi.contract.UnpackLog(event, "LogGuardianSetChanged", log); err != nil {
		return nil, err
	}
	return event, nil
}

// AbiLogTokensLockedIterator is returned from FilterLogTokensLocked and is used to iterate over the raw logs and unpacked data for LogTokensLocked events raised by the Abi contract.
type AbiLogTokensLockedIterator struct {
	Event *AbiLogTokensLocked // Event containing the contract specifics and raw log

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
func (it *AbiLogTokensLockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiLogTokensLocked)
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
		it.Event = new(AbiLogTokensLocked)
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
func (it *AbiLogTokensLockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiLogTokensLockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiLogTokensLocked represents a LogTokensLocked event raised by the Abi contract.
type AbiLogTokensLocked struct {
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
func (_Abi *AbiFilterer) FilterLogTokensLocked(opts *bind.FilterOpts, token [][32]byte, sender [][32]byte) (*AbiLogTokensLockedIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Abi.contract.FilterLogs(opts, "LogTokensLocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &AbiLogTokensLockedIterator{contract: _Abi.contract, event: "LogTokensLocked", logs: logs, sub: sub}, nil
}

// WatchLogTokensLocked is a free log subscription operation binding the contract event 0x84b445260a99044cc9529b3033663c078031a14e31f3c255ff02c62667bab14b.
//
// Solidity: event LogTokensLocked(uint8 target_chain, uint8 token_chain, bytes32 indexed token, bytes32 indexed sender, bytes32 recipient, uint256 amount)
func (_Abi *AbiFilterer) WatchLogTokensLocked(opts *bind.WatchOpts, sink chan<- *AbiLogTokensLocked, token [][32]byte, sender [][32]byte) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Abi.contract.WatchLogs(opts, "LogTokensLocked", tokenRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiLogTokensLocked)
				if err := _Abi.contract.UnpackLog(event, "LogTokensLocked", log); err != nil {
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
func (_Abi *AbiFilterer) ParseLogTokensLocked(log types.Log) (*AbiLogTokensLocked, error) {
	event := new(AbiLogTokensLocked)
	if err := _Abi.contract.UnpackLog(event, "LogTokensLocked", log); err != nil {
		return nil, err
	}
	return event, nil
}

