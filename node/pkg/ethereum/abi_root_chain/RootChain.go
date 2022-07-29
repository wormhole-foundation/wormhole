// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi_root_chain

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

// AbiRootChainABI is the input ABI used to generate the binding from.
const AbiRootChainABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"headerBlockId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"start\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"end\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"}],\"name\":\"NewHeaderBlock\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"headerBlockId\",\"type\":\"uint256\"}],\"name\":\"ResetHeaderBlock\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[],\"name\":\"CHAINID\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"VOTE_TYPE\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"_nextHeaderBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"currentHeaderBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getLastChildBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"headerBlocks\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"start\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"end\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdAt\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"proposer\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"heimdallId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"networkId\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"string\",\"name\":\"_heimdallId\",\"type\":\"string\"}],\"name\":\"setHeimdallId\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"setNextHeaderBlock\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"slash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256[3][]\",\"name\":\"sigs\",\"type\":\"uint256[3][]\"}],\"name\":\"submitCheckpoint\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"sigs\",\"type\":\"bytes\"}],\"name\":\"submitHeaderBlock\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"numDeposits\",\"type\":\"uint256\"}],\"name\":\"updateDepositId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"depositId\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// AbiRootChain is an auto generated Go binding around an Ethereum contract.
type AbiRootChain struct {
	AbiRootChainCaller     // Read-only binding to the contract
	AbiRootChainTransactor // Write-only binding to the contract
	AbiRootChainFilterer   // Log filterer for contract events
}

// AbiRootChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type AbiRootChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiRootChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AbiRootChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiRootChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AbiRootChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiRootChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AbiRootChainSession struct {
	Contract     *AbiRootChain     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AbiRootChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AbiRootChainCallerSession struct {
	Contract *AbiRootChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// AbiRootChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AbiRootChainTransactorSession struct {
	Contract     *AbiRootChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// AbiRootChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type AbiRootChainRaw struct {
	Contract *AbiRootChain // Generic contract binding to access the raw methods on
}

// AbiRootChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AbiRootChainCallerRaw struct {
	Contract *AbiRootChainCaller // Generic read-only contract binding to access the raw methods on
}

// AbiRootChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AbiRootChainTransactorRaw struct {
	Contract *AbiRootChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAbiRootChain creates a new instance of AbiRootChain, bound to a specific deployed contract.
func NewAbiRootChain(address common.Address, backend bind.ContractBackend) (*AbiRootChain, error) {
	contract, err := bindAbiRootChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AbiRootChain{AbiRootChainCaller: AbiRootChainCaller{contract: contract}, AbiRootChainTransactor: AbiRootChainTransactor{contract: contract}, AbiRootChainFilterer: AbiRootChainFilterer{contract: contract}}, nil
}

// NewAbiRootChainCaller creates a new read-only instance of AbiRootChain, bound to a specific deployed contract.
func NewAbiRootChainCaller(address common.Address, caller bind.ContractCaller) (*AbiRootChainCaller, error) {
	contract, err := bindAbiRootChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainCaller{contract: contract}, nil
}

// NewAbiRootChainTransactor creates a new write-only instance of AbiRootChain, bound to a specific deployed contract.
func NewAbiRootChainTransactor(address common.Address, transactor bind.ContractTransactor) (*AbiRootChainTransactor, error) {
	contract, err := bindAbiRootChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainTransactor{contract: contract}, nil
}

// NewAbiRootChainFilterer creates a new log filterer instance of AbiRootChain, bound to a specific deployed contract.
func NewAbiRootChainFilterer(address common.Address, filterer bind.ContractFilterer) (*AbiRootChainFilterer, error) {
	contract, err := bindAbiRootChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainFilterer{contract: contract}, nil
}

// bindAbiRootChain binds a generic wrapper to an already deployed contract.
func bindAbiRootChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AbiRootChainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbiRootChain *AbiRootChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbiRootChain.Contract.AbiRootChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbiRootChain *AbiRootChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiRootChain.Contract.AbiRootChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbiRootChain *AbiRootChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbiRootChain.Contract.AbiRootChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbiRootChain *AbiRootChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbiRootChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbiRootChain *AbiRootChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiRootChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbiRootChain *AbiRootChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbiRootChain.Contract.contract.Transact(opts, method, params...)
}

// CHAINID is a free data retrieval call binding the contract method 0xcc79f97b.
//
// Solidity: function CHAINID() view returns(uint256)
func (_AbiRootChain *AbiRootChainCaller) CHAINID(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "CHAINID")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CHAINID is a free data retrieval call binding the contract method 0xcc79f97b.
//
// Solidity: function CHAINID() view returns(uint256)
func (_AbiRootChain *AbiRootChainSession) CHAINID() (*big.Int, error) {
	return _AbiRootChain.Contract.CHAINID(&_AbiRootChain.CallOpts)
}

// CHAINID is a free data retrieval call binding the contract method 0xcc79f97b.
//
// Solidity: function CHAINID() view returns(uint256)
func (_AbiRootChain *AbiRootChainCallerSession) CHAINID() (*big.Int, error) {
	return _AbiRootChain.Contract.CHAINID(&_AbiRootChain.CallOpts)
}

// VOTETYPE is a free data retrieval call binding the contract method 0xd5b844eb.
//
// Solidity: function VOTE_TYPE() view returns(uint8)
func (_AbiRootChain *AbiRootChainCaller) VOTETYPE(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "VOTE_TYPE")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// VOTETYPE is a free data retrieval call binding the contract method 0xd5b844eb.
//
// Solidity: function VOTE_TYPE() view returns(uint8)
func (_AbiRootChain *AbiRootChainSession) VOTETYPE() (uint8, error) {
	return _AbiRootChain.Contract.VOTETYPE(&_AbiRootChain.CallOpts)
}

// VOTETYPE is a free data retrieval call binding the contract method 0xd5b844eb.
//
// Solidity: function VOTE_TYPE() view returns(uint8)
func (_AbiRootChain *AbiRootChainCallerSession) VOTETYPE() (uint8, error) {
	return _AbiRootChain.Contract.VOTETYPE(&_AbiRootChain.CallOpts)
}

// NextHeaderBlock is a free data retrieval call binding the contract method 0x8d978d88.
//
// Solidity: function _nextHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCaller) NextHeaderBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "_nextHeaderBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextHeaderBlock is a free data retrieval call binding the contract method 0x8d978d88.
//
// Solidity: function _nextHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainSession) NextHeaderBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.NextHeaderBlock(&_AbiRootChain.CallOpts)
}

// NextHeaderBlock is a free data retrieval call binding the contract method 0x8d978d88.
//
// Solidity: function _nextHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCallerSession) NextHeaderBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.NextHeaderBlock(&_AbiRootChain.CallOpts)
}

// CurrentHeaderBlock is a free data retrieval call binding the contract method 0xec7e4855.
//
// Solidity: function currentHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCaller) CurrentHeaderBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "currentHeaderBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentHeaderBlock is a free data retrieval call binding the contract method 0xec7e4855.
//
// Solidity: function currentHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainSession) CurrentHeaderBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.CurrentHeaderBlock(&_AbiRootChain.CallOpts)
}

// CurrentHeaderBlock is a free data retrieval call binding the contract method 0xec7e4855.
//
// Solidity: function currentHeaderBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCallerSession) CurrentHeaderBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.CurrentHeaderBlock(&_AbiRootChain.CallOpts)
}

// GetLastChildBlock is a free data retrieval call binding the contract method 0xb87e1b66.
//
// Solidity: function getLastChildBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCaller) GetLastChildBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "getLastChildBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastChildBlock is a free data retrieval call binding the contract method 0xb87e1b66.
//
// Solidity: function getLastChildBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainSession) GetLastChildBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.GetLastChildBlock(&_AbiRootChain.CallOpts)
}

// GetLastChildBlock is a free data retrieval call binding the contract method 0xb87e1b66.
//
// Solidity: function getLastChildBlock() view returns(uint256)
func (_AbiRootChain *AbiRootChainCallerSession) GetLastChildBlock() (*big.Int, error) {
	return _AbiRootChain.Contract.GetLastChildBlock(&_AbiRootChain.CallOpts)
}

// HeaderBlocks is a free data retrieval call binding the contract method 0x41539d4a.
//
// Solidity: function headerBlocks(uint256 ) view returns(bytes32 root, uint256 start, uint256 end, uint256 createdAt, address proposer)
func (_AbiRootChain *AbiRootChainCaller) HeaderBlocks(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Root      [32]byte
	Start     *big.Int
	End       *big.Int
	CreatedAt *big.Int
	Proposer  common.Address
}, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "headerBlocks", arg0)

	outstruct := new(struct {
		Root      [32]byte
		Start     *big.Int
		End       *big.Int
		CreatedAt *big.Int
		Proposer  common.Address
	})

	outstruct.Root = out[0].([32]byte)
	outstruct.Start = out[1].(*big.Int)
	outstruct.End = out[2].(*big.Int)
	outstruct.CreatedAt = out[3].(*big.Int)
	outstruct.Proposer = out[4].(common.Address)

	return *outstruct, err

}

// HeaderBlocks is a free data retrieval call binding the contract method 0x41539d4a.
//
// Solidity: function headerBlocks(uint256 ) view returns(bytes32 root, uint256 start, uint256 end, uint256 createdAt, address proposer)
func (_AbiRootChain *AbiRootChainSession) HeaderBlocks(arg0 *big.Int) (struct {
	Root      [32]byte
	Start     *big.Int
	End       *big.Int
	CreatedAt *big.Int
	Proposer  common.Address
}, error) {
	return _AbiRootChain.Contract.HeaderBlocks(&_AbiRootChain.CallOpts, arg0)
}

// HeaderBlocks is a free data retrieval call binding the contract method 0x41539d4a.
//
// Solidity: function headerBlocks(uint256 ) view returns(bytes32 root, uint256 start, uint256 end, uint256 createdAt, address proposer)
func (_AbiRootChain *AbiRootChainCallerSession) HeaderBlocks(arg0 *big.Int) (struct {
	Root      [32]byte
	Start     *big.Int
	End       *big.Int
	CreatedAt *big.Int
	Proposer  common.Address
}, error) {
	return _AbiRootChain.Contract.HeaderBlocks(&_AbiRootChain.CallOpts, arg0)
}

// HeimdallId is a free data retrieval call binding the contract method 0xfbc3dd36.
//
// Solidity: function heimdallId() view returns(bytes32)
func (_AbiRootChain *AbiRootChainCaller) HeimdallId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "heimdallId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// HeimdallId is a free data retrieval call binding the contract method 0xfbc3dd36.
//
// Solidity: function heimdallId() view returns(bytes32)
func (_AbiRootChain *AbiRootChainSession) HeimdallId() ([32]byte, error) {
	return _AbiRootChain.Contract.HeimdallId(&_AbiRootChain.CallOpts)
}

// HeimdallId is a free data retrieval call binding the contract method 0xfbc3dd36.
//
// Solidity: function heimdallId() view returns(bytes32)
func (_AbiRootChain *AbiRootChainCallerSession) HeimdallId() ([32]byte, error) {
	return _AbiRootChain.Contract.HeimdallId(&_AbiRootChain.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_AbiRootChain *AbiRootChainCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "isOwner")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_AbiRootChain *AbiRootChainSession) IsOwner() (bool, error) {
	return _AbiRootChain.Contract.IsOwner(&_AbiRootChain.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_AbiRootChain *AbiRootChainCallerSession) IsOwner() (bool, error) {
	return _AbiRootChain.Contract.IsOwner(&_AbiRootChain.CallOpts)
}

// NetworkId is a free data retrieval call binding the contract method 0x9025e64c.
//
// Solidity: function networkId() view returns(bytes)
func (_AbiRootChain *AbiRootChainCaller) NetworkId(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "networkId")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// NetworkId is a free data retrieval call binding the contract method 0x9025e64c.
//
// Solidity: function networkId() view returns(bytes)
func (_AbiRootChain *AbiRootChainSession) NetworkId() ([]byte, error) {
	return _AbiRootChain.Contract.NetworkId(&_AbiRootChain.CallOpts)
}

// NetworkId is a free data retrieval call binding the contract method 0x9025e64c.
//
// Solidity: function networkId() view returns(bytes)
func (_AbiRootChain *AbiRootChainCallerSession) NetworkId() ([]byte, error) {
	return _AbiRootChain.Contract.NetworkId(&_AbiRootChain.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AbiRootChain *AbiRootChainCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AbiRootChain.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AbiRootChain *AbiRootChainSession) Owner() (common.Address, error) {
	return _AbiRootChain.Contract.Owner(&_AbiRootChain.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AbiRootChain *AbiRootChainCallerSession) Owner() (common.Address, error) {
	return _AbiRootChain.Contract.Owner(&_AbiRootChain.CallOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AbiRootChain *AbiRootChainTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AbiRootChain *AbiRootChainSession) RenounceOwnership() (*types.Transaction, error) {
	return _AbiRootChain.Contract.RenounceOwnership(&_AbiRootChain.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AbiRootChain *AbiRootChainTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _AbiRootChain.Contract.RenounceOwnership(&_AbiRootChain.TransactOpts)
}

// SetHeimdallId is a paid mutator transaction binding the contract method 0xea0688b3.
//
// Solidity: function setHeimdallId(string _heimdallId) returns()
func (_AbiRootChain *AbiRootChainTransactor) SetHeimdallId(opts *bind.TransactOpts, _heimdallId string) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "setHeimdallId", _heimdallId)
}

// SetHeimdallId is a paid mutator transaction binding the contract method 0xea0688b3.
//
// Solidity: function setHeimdallId(string _heimdallId) returns()
func (_AbiRootChain *AbiRootChainSession) SetHeimdallId(_heimdallId string) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SetHeimdallId(&_AbiRootChain.TransactOpts, _heimdallId)
}

// SetHeimdallId is a paid mutator transaction binding the contract method 0xea0688b3.
//
// Solidity: function setHeimdallId(string _heimdallId) returns()
func (_AbiRootChain *AbiRootChainTransactorSession) SetHeimdallId(_heimdallId string) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SetHeimdallId(&_AbiRootChain.TransactOpts, _heimdallId)
}

// SetNextHeaderBlock is a paid mutator transaction binding the contract method 0xcf24a0ea.
//
// Solidity: function setNextHeaderBlock(uint256 _value) returns()
func (_AbiRootChain *AbiRootChainTransactor) SetNextHeaderBlock(opts *bind.TransactOpts, _value *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "setNextHeaderBlock", _value)
}

// SetNextHeaderBlock is a paid mutator transaction binding the contract method 0xcf24a0ea.
//
// Solidity: function setNextHeaderBlock(uint256 _value) returns()
func (_AbiRootChain *AbiRootChainSession) SetNextHeaderBlock(_value *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SetNextHeaderBlock(&_AbiRootChain.TransactOpts, _value)
}

// SetNextHeaderBlock is a paid mutator transaction binding the contract method 0xcf24a0ea.
//
// Solidity: function setNextHeaderBlock(uint256 _value) returns()
func (_AbiRootChain *AbiRootChainTransactorSession) SetNextHeaderBlock(_value *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SetNextHeaderBlock(&_AbiRootChain.TransactOpts, _value)
}

// Slash is a paid mutator transaction binding the contract method 0x2da25de3.
//
// Solidity: function slash() returns()
func (_AbiRootChain *AbiRootChainTransactor) Slash(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "slash")
}

// Slash is a paid mutator transaction binding the contract method 0x2da25de3.
//
// Solidity: function slash() returns()
func (_AbiRootChain *AbiRootChainSession) Slash() (*types.Transaction, error) {
	return _AbiRootChain.Contract.Slash(&_AbiRootChain.TransactOpts)
}

// Slash is a paid mutator transaction binding the contract method 0x2da25de3.
//
// Solidity: function slash() returns()
func (_AbiRootChain *AbiRootChainTransactorSession) Slash() (*types.Transaction, error) {
	return _AbiRootChain.Contract.Slash(&_AbiRootChain.TransactOpts)
}

// SubmitCheckpoint is a paid mutator transaction binding the contract method 0x4e43e495.
//
// Solidity: function submitCheckpoint(bytes data, uint256[3][] sigs) returns()
func (_AbiRootChain *AbiRootChainTransactor) SubmitCheckpoint(opts *bind.TransactOpts, data []byte, sigs [][3]*big.Int) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "submitCheckpoint", data, sigs)
}

// SubmitCheckpoint is a paid mutator transaction binding the contract method 0x4e43e495.
//
// Solidity: function submitCheckpoint(bytes data, uint256[3][] sigs) returns()
func (_AbiRootChain *AbiRootChainSession) SubmitCheckpoint(data []byte, sigs [][3]*big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SubmitCheckpoint(&_AbiRootChain.TransactOpts, data, sigs)
}

// SubmitCheckpoint is a paid mutator transaction binding the contract method 0x4e43e495.
//
// Solidity: function submitCheckpoint(bytes data, uint256[3][] sigs) returns()
func (_AbiRootChain *AbiRootChainTransactorSession) SubmitCheckpoint(data []byte, sigs [][3]*big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SubmitCheckpoint(&_AbiRootChain.TransactOpts, data, sigs)
}

// SubmitHeaderBlock is a paid mutator transaction binding the contract method 0x6a791f11.
//
// Solidity: function submitHeaderBlock(bytes data, bytes sigs) returns()
func (_AbiRootChain *AbiRootChainTransactor) SubmitHeaderBlock(opts *bind.TransactOpts, data []byte, sigs []byte) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "submitHeaderBlock", data, sigs)
}

// SubmitHeaderBlock is a paid mutator transaction binding the contract method 0x6a791f11.
//
// Solidity: function submitHeaderBlock(bytes data, bytes sigs) returns()
func (_AbiRootChain *AbiRootChainSession) SubmitHeaderBlock(data []byte, sigs []byte) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SubmitHeaderBlock(&_AbiRootChain.TransactOpts, data, sigs)
}

// SubmitHeaderBlock is a paid mutator transaction binding the contract method 0x6a791f11.
//
// Solidity: function submitHeaderBlock(bytes data, bytes sigs) returns()
func (_AbiRootChain *AbiRootChainTransactorSession) SubmitHeaderBlock(data []byte, sigs []byte) (*types.Transaction, error) {
	return _AbiRootChain.Contract.SubmitHeaderBlock(&_AbiRootChain.TransactOpts, data, sigs)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AbiRootChain *AbiRootChainTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AbiRootChain *AbiRootChainSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AbiRootChain.Contract.TransferOwnership(&_AbiRootChain.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AbiRootChain *AbiRootChainTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AbiRootChain.Contract.TransferOwnership(&_AbiRootChain.TransactOpts, newOwner)
}

// UpdateDepositId is a paid mutator transaction binding the contract method 0x5391f483.
//
// Solidity: function updateDepositId(uint256 numDeposits) returns(uint256 depositId)
func (_AbiRootChain *AbiRootChainTransactor) UpdateDepositId(opts *bind.TransactOpts, numDeposits *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.contract.Transact(opts, "updateDepositId", numDeposits)
}

// UpdateDepositId is a paid mutator transaction binding the contract method 0x5391f483.
//
// Solidity: function updateDepositId(uint256 numDeposits) returns(uint256 depositId)
func (_AbiRootChain *AbiRootChainSession) UpdateDepositId(numDeposits *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.UpdateDepositId(&_AbiRootChain.TransactOpts, numDeposits)
}

// UpdateDepositId is a paid mutator transaction binding the contract method 0x5391f483.
//
// Solidity: function updateDepositId(uint256 numDeposits) returns(uint256 depositId)
func (_AbiRootChain *AbiRootChainTransactorSession) UpdateDepositId(numDeposits *big.Int) (*types.Transaction, error) {
	return _AbiRootChain.Contract.UpdateDepositId(&_AbiRootChain.TransactOpts, numDeposits)
}

// AbiRootChainNewHeaderBlockIterator is returned from FilterNewHeaderBlock and is used to iterate over the raw logs and unpacked data for NewHeaderBlock events raised by the AbiRootChain contract.
type AbiRootChainNewHeaderBlockIterator struct {
	Event *AbiRootChainNewHeaderBlock // Event containing the contract specifics and raw log

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
func (it *AbiRootChainNewHeaderBlockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiRootChainNewHeaderBlock)
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
		it.Event = new(AbiRootChainNewHeaderBlock)
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
func (it *AbiRootChainNewHeaderBlockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiRootChainNewHeaderBlockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiRootChainNewHeaderBlock represents a NewHeaderBlock event raised by the AbiRootChain contract.
type AbiRootChainNewHeaderBlock struct {
	Proposer      common.Address
	HeaderBlockId *big.Int
	Reward        *big.Int
	Start         *big.Int
	End           *big.Int
	Root          [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewHeaderBlock is a free log retrieval operation binding the contract event 0xba5de06d22af2685c6c7765f60067f7d2b08c2d29f53cdf14d67f6d1c9bfb527.
//
// Solidity: event NewHeaderBlock(address indexed proposer, uint256 indexed headerBlockId, uint256 indexed reward, uint256 start, uint256 end, bytes32 root)
func (_AbiRootChain *AbiRootChainFilterer) FilterNewHeaderBlock(opts *bind.FilterOpts, proposer []common.Address, headerBlockId []*big.Int, reward []*big.Int) (*AbiRootChainNewHeaderBlockIterator, error) {

	var proposerRule []interface{}
	for _, proposerItem := range proposer {
		proposerRule = append(proposerRule, proposerItem)
	}
	var headerBlockIdRule []interface{}
	for _, headerBlockIdItem := range headerBlockId {
		headerBlockIdRule = append(headerBlockIdRule, headerBlockIdItem)
	}
	var rewardRule []interface{}
	for _, rewardItem := range reward {
		rewardRule = append(rewardRule, rewardItem)
	}

	logs, sub, err := _AbiRootChain.contract.FilterLogs(opts, "NewHeaderBlock", proposerRule, headerBlockIdRule, rewardRule)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainNewHeaderBlockIterator{contract: _AbiRootChain.contract, event: "NewHeaderBlock", logs: logs, sub: sub}, nil
}

// WatchNewHeaderBlock is a free log subscription operation binding the contract event 0xba5de06d22af2685c6c7765f60067f7d2b08c2d29f53cdf14d67f6d1c9bfb527.
//
// Solidity: event NewHeaderBlock(address indexed proposer, uint256 indexed headerBlockId, uint256 indexed reward, uint256 start, uint256 end, bytes32 root)
func (_AbiRootChain *AbiRootChainFilterer) WatchNewHeaderBlock(opts *bind.WatchOpts, sink chan<- *AbiRootChainNewHeaderBlock, proposer []common.Address, headerBlockId []*big.Int, reward []*big.Int) (event.Subscription, error) {

	var proposerRule []interface{}
	for _, proposerItem := range proposer {
		proposerRule = append(proposerRule, proposerItem)
	}
	var headerBlockIdRule []interface{}
	for _, headerBlockIdItem := range headerBlockId {
		headerBlockIdRule = append(headerBlockIdRule, headerBlockIdItem)
	}
	var rewardRule []interface{}
	for _, rewardItem := range reward {
		rewardRule = append(rewardRule, rewardItem)
	}

	logs, sub, err := _AbiRootChain.contract.WatchLogs(opts, "NewHeaderBlock", proposerRule, headerBlockIdRule, rewardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiRootChainNewHeaderBlock)
				if err := _AbiRootChain.contract.UnpackLog(event, "NewHeaderBlock", log); err != nil {
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

// ParseNewHeaderBlock is a log parse operation binding the contract event 0xba5de06d22af2685c6c7765f60067f7d2b08c2d29f53cdf14d67f6d1c9bfb527.
//
// Solidity: event NewHeaderBlock(address indexed proposer, uint256 indexed headerBlockId, uint256 indexed reward, uint256 start, uint256 end, bytes32 root)
func (_AbiRootChain *AbiRootChainFilterer) ParseNewHeaderBlock(log types.Log) (*AbiRootChainNewHeaderBlock, error) {
	event := new(AbiRootChainNewHeaderBlock)
	if err := _AbiRootChain.contract.UnpackLog(event, "NewHeaderBlock", log); err != nil {
		return nil, err
	}
	return event, nil
}

// AbiRootChainOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the AbiRootChain contract.
type AbiRootChainOwnershipTransferredIterator struct {
	Event *AbiRootChainOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *AbiRootChainOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiRootChainOwnershipTransferred)
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
		it.Event = new(AbiRootChainOwnershipTransferred)
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
func (it *AbiRootChainOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiRootChainOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiRootChainOwnershipTransferred represents a OwnershipTransferred event raised by the AbiRootChain contract.
type AbiRootChainOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AbiRootChain *AbiRootChainFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AbiRootChainOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AbiRootChain.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainOwnershipTransferredIterator{contract: _AbiRootChain.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AbiRootChain *AbiRootChainFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *AbiRootChainOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AbiRootChain.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiRootChainOwnershipTransferred)
				if err := _AbiRootChain.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AbiRootChain *AbiRootChainFilterer) ParseOwnershipTransferred(log types.Log) (*AbiRootChainOwnershipTransferred, error) {
	event := new(AbiRootChainOwnershipTransferred)
	if err := _AbiRootChain.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	return event, nil
}

// AbiRootChainResetHeaderBlockIterator is returned from FilterResetHeaderBlock and is used to iterate over the raw logs and unpacked data for ResetHeaderBlock events raised by the AbiRootChain contract.
type AbiRootChainResetHeaderBlockIterator struct {
	Event *AbiRootChainResetHeaderBlock // Event containing the contract specifics and raw log

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
func (it *AbiRootChainResetHeaderBlockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbiRootChainResetHeaderBlock)
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
		it.Event = new(AbiRootChainResetHeaderBlock)
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
func (it *AbiRootChainResetHeaderBlockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbiRootChainResetHeaderBlockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbiRootChainResetHeaderBlock represents a ResetHeaderBlock event raised by the AbiRootChain contract.
type AbiRootChainResetHeaderBlock struct {
	Proposer      common.Address
	HeaderBlockId *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterResetHeaderBlock is a free log retrieval operation binding the contract event 0xca1d8316287f938830e225956a7bb10fd5a1a1506dd2eb3a476751a488117205.
//
// Solidity: event ResetHeaderBlock(address indexed proposer, uint256 indexed headerBlockId)
func (_AbiRootChain *AbiRootChainFilterer) FilterResetHeaderBlock(opts *bind.FilterOpts, proposer []common.Address, headerBlockId []*big.Int) (*AbiRootChainResetHeaderBlockIterator, error) {

	var proposerRule []interface{}
	for _, proposerItem := range proposer {
		proposerRule = append(proposerRule, proposerItem)
	}
	var headerBlockIdRule []interface{}
	for _, headerBlockIdItem := range headerBlockId {
		headerBlockIdRule = append(headerBlockIdRule, headerBlockIdItem)
	}

	logs, sub, err := _AbiRootChain.contract.FilterLogs(opts, "ResetHeaderBlock", proposerRule, headerBlockIdRule)
	if err != nil {
		return nil, err
	}
	return &AbiRootChainResetHeaderBlockIterator{contract: _AbiRootChain.contract, event: "ResetHeaderBlock", logs: logs, sub: sub}, nil
}

// WatchResetHeaderBlock is a free log subscription operation binding the contract event 0xca1d8316287f938830e225956a7bb10fd5a1a1506dd2eb3a476751a488117205.
//
// Solidity: event ResetHeaderBlock(address indexed proposer, uint256 indexed headerBlockId)
func (_AbiRootChain *AbiRootChainFilterer) WatchResetHeaderBlock(opts *bind.WatchOpts, sink chan<- *AbiRootChainResetHeaderBlock, proposer []common.Address, headerBlockId []*big.Int) (event.Subscription, error) {

	var proposerRule []interface{}
	for _, proposerItem := range proposer {
		proposerRule = append(proposerRule, proposerItem)
	}
	var headerBlockIdRule []interface{}
	for _, headerBlockIdItem := range headerBlockId {
		headerBlockIdRule = append(headerBlockIdRule, headerBlockIdItem)
	}

	logs, sub, err := _AbiRootChain.contract.WatchLogs(opts, "ResetHeaderBlock", proposerRule, headerBlockIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiRootChainResetHeaderBlock)
				if err := _AbiRootChain.contract.UnpackLog(event, "ResetHeaderBlock", log); err != nil {
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

// ParseResetHeaderBlock is a log parse operation binding the contract event 0xca1d8316287f938830e225956a7bb10fd5a1a1506dd2eb3a476751a488117205.
//
// Solidity: event ResetHeaderBlock(address indexed proposer, uint256 indexed headerBlockId)
func (_AbiRootChain *AbiRootChainFilterer) ParseResetHeaderBlock(log types.Log) (*AbiRootChainResetHeaderBlock, error) {
	event := new(AbiRootChainResetHeaderBlock)
	if err := _AbiRootChain.contract.UnpackLog(event, "ResetHeaderBlock", log); err != nil {
		return nil, err
	}
	return event, nil
}
