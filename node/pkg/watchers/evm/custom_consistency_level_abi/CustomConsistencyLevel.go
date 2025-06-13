// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ccl

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

// CclMetaData contains all meta data concerning the Ccl contract.
var CclMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configure\",\"inputs\":[{\"name\":\"config\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getConfiguration\",\"inputs\":[{\"name\":\"emitterAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ConfigSet\",\"inputs\":[{\"name\":\"emitterAddress\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"config\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false}]",
}

// CclABI is the input ABI used to generate the binding from.
// Deprecated: Use CclMetaData.ABI instead.
var CclABI = CclMetaData.ABI

// Ccl is an auto generated Go binding around an Ethereum contract.
type Ccl struct {
	CclCaller     // Read-only binding to the contract
	CclTransactor // Write-only binding to the contract
	CclFilterer   // Log filterer for contract events
}

// CclCaller is an auto generated read-only Go binding around an Ethereum contract.
type CclCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CclTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CclTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CclFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CclFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CclSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CclSession struct {
	Contract     *Ccl              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CclCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CclCallerSession struct {
	Contract *CclCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// CclTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CclTransactorSession struct {
	Contract     *CclTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CclRaw is an auto generated low-level Go binding around an Ethereum contract.
type CclRaw struct {
	Contract *Ccl // Generic contract binding to access the raw methods on
}

// CclCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CclCallerRaw struct {
	Contract *CclCaller // Generic read-only contract binding to access the raw methods on
}

// CclTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CclTransactorRaw struct {
	Contract *CclTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCcl creates a new instance of Ccl, bound to a specific deployed contract.
func NewCcl(address common.Address, backend bind.ContractBackend) (*Ccl, error) {
	contract, err := bindCcl(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Ccl{CclCaller: CclCaller{contract: contract}, CclTransactor: CclTransactor{contract: contract}, CclFilterer: CclFilterer{contract: contract}}, nil
}

// NewCclCaller creates a new read-only instance of Ccl, bound to a specific deployed contract.
func NewCclCaller(address common.Address, caller bind.ContractCaller) (*CclCaller, error) {
	contract, err := bindCcl(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CclCaller{contract: contract}, nil
}

// NewCclTransactor creates a new write-only instance of Ccl, bound to a specific deployed contract.
func NewCclTransactor(address common.Address, transactor bind.ContractTransactor) (*CclTransactor, error) {
	contract, err := bindCcl(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CclTransactor{contract: contract}, nil
}

// NewCclFilterer creates a new log filterer instance of Ccl, bound to a specific deployed contract.
func NewCclFilterer(address common.Address, filterer bind.ContractFilterer) (*CclFilterer, error) {
	contract, err := bindCcl(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CclFilterer{contract: contract}, nil
}

// bindCcl binds a generic wrapper to an already deployed contract.
func bindCcl(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CclMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ccl *CclRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ccl.Contract.CclCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ccl *CclRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ccl.Contract.CclTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ccl *CclRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ccl.Contract.CclTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ccl *CclCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ccl.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ccl *CclTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ccl.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ccl *CclTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ccl.Contract.contract.Transact(opts, method, params...)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_Ccl *CclCaller) VERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Ccl.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_Ccl *CclSession) VERSION() (string, error) {
	return _Ccl.Contract.VERSION(&_Ccl.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_Ccl *CclCallerSession) VERSION() (string, error) {
	return _Ccl.Contract.VERSION(&_Ccl.CallOpts)
}

// GetConfiguration is a free data retrieval call binding the contract method 0xc44b11f7.
//
// Solidity: function getConfiguration(address emitterAddress) view returns(bytes32)
func (_Ccl *CclCaller) GetConfiguration(opts *bind.CallOpts, emitterAddress common.Address) ([32]byte, error) {
	var out []interface{}
	err := _Ccl.contract.Call(opts, &out, "getConfiguration", emitterAddress)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetConfiguration is a free data retrieval call binding the contract method 0xc44b11f7.
//
// Solidity: function getConfiguration(address emitterAddress) view returns(bytes32)
func (_Ccl *CclSession) GetConfiguration(emitterAddress common.Address) ([32]byte, error) {
	return _Ccl.Contract.GetConfiguration(&_Ccl.CallOpts, emitterAddress)
}

// GetConfiguration is a free data retrieval call binding the contract method 0xc44b11f7.
//
// Solidity: function getConfiguration(address emitterAddress) view returns(bytes32)
func (_Ccl *CclCallerSession) GetConfiguration(emitterAddress common.Address) ([32]byte, error) {
	return _Ccl.Contract.GetConfiguration(&_Ccl.CallOpts, emitterAddress)
}

// Configure is a paid mutator transaction binding the contract method 0xca23addf.
//
// Solidity: function configure(bytes32 config) returns()
func (_Ccl *CclTransactor) Configure(opts *bind.TransactOpts, config [32]byte) (*types.Transaction, error) {
	return _Ccl.contract.Transact(opts, "configure", config)
}

// Configure is a paid mutator transaction binding the contract method 0xca23addf.
//
// Solidity: function configure(bytes32 config) returns()
func (_Ccl *CclSession) Configure(config [32]byte) (*types.Transaction, error) {
	return _Ccl.Contract.Configure(&_Ccl.TransactOpts, config)
}

// Configure is a paid mutator transaction binding the contract method 0xca23addf.
//
// Solidity: function configure(bytes32 config) returns()
func (_Ccl *CclTransactorSession) Configure(config [32]byte) (*types.Transaction, error) {
	return _Ccl.Contract.Configure(&_Ccl.TransactOpts, config)
}

// CclConfigSetIterator is returned from FilterConfigSet and is used to iterate over the raw logs and unpacked data for ConfigSet events raised by the Ccl contract.
type CclConfigSetIterator struct {
	Event *CclConfigSet // Event containing the contract specifics and raw log

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
func (it *CclConfigSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CclConfigSet)
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
		it.Event = new(CclConfigSet)
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
func (it *CclConfigSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CclConfigSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CclConfigSet represents a ConfigSet event raised by the Ccl contract.
type CclConfigSet struct {
	EmitterAddress common.Address
	Config         [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterConfigSet is a free log retrieval operation binding the contract event 0xa37f0112e03d41de27266c1680238ff1548c0441ad1e73c82917c000eefdd5ea.
//
// Solidity: event ConfigSet(address emitterAddress, bytes32 config)
func (_Ccl *CclFilterer) FilterConfigSet(opts *bind.FilterOpts) (*CclConfigSetIterator, error) {

	logs, sub, err := _Ccl.contract.FilterLogs(opts, "ConfigSet")
	if err != nil {
		return nil, err
	}
	return &CclConfigSetIterator{contract: _Ccl.contract, event: "ConfigSet", logs: logs, sub: sub}, nil
}

// WatchConfigSet is a free log subscription operation binding the contract event 0xa37f0112e03d41de27266c1680238ff1548c0441ad1e73c82917c000eefdd5ea.
//
// Solidity: event ConfigSet(address emitterAddress, bytes32 config)
func (_Ccl *CclFilterer) WatchConfigSet(opts *bind.WatchOpts, sink chan<- *CclConfigSet) (event.Subscription, error) {

	logs, sub, err := _Ccl.contract.WatchLogs(opts, "ConfigSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CclConfigSet)
				if err := _Ccl.contract.UnpackLog(event, "ConfigSet", log); err != nil {
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

// ParseConfigSet is a log parse operation binding the contract event 0xa37f0112e03d41de27266c1680238ff1548c0441ad1e73c82917c000eefdd5ea.
//
// Solidity: event ConfigSet(address emitterAddress, bytes32 config)
func (_Ccl *CclFilterer) ParseConfigSet(log types.Log) (*CclConfigSet, error) {
	event := new(CclConfigSet)
	if err := _Ccl.contract.UnpackLog(event, "ConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
