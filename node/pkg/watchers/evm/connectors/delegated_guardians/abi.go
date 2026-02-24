// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package delegated_guardians

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

// WormholeDelegatedGuardiansDelegatedGuardianSet is an auto generated low-level Go binding around an user-defined struct.
type WormholeDelegatedGuardiansDelegatedGuardianSet struct {
	ChainId   uint16
	Timestamp uint32
	Threshold uint8
	Keys      []common.Address
}

// DelegatedguardiansMetaData contains all meta data concerning the Delegatedguardians contract.
var DelegatedguardiansMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wormholeAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"chainIdsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainIds\",\"outputs\":[{\"internalType\":\"uint16[]\",\"name\":\"\",\"type\":\"uint16[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_chainId\",\"type\":\"uint16\"}],\"name\":\"getConfig\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"threshold\",\"type\":\"uint8\"},{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"}],\"internalType\":\"structWormholeDelegatedGuardians.DelegatedGuardianSet\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getConfig\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"threshold\",\"type\":\"uint8\"},{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"}],\"internalType\":\"structWormholeDelegatedGuardians.DelegatedGuardianSet[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_chainId\",\"type\":\"uint16\"}],\"name\":\"getHistoricalConfig\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"threshold\",\"type\":\"uint8\"},{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"}],\"internalType\":\"structWormholeDelegatedGuardians.DelegatedGuardianSet[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getHistoricalConfig\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"timestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"threshold\",\"type\":\"uint8\"},{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"}],\"internalType\":\"structWormholeDelegatedGuardians.DelegatedGuardianSet\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_chainId\",\"type\":\"uint16\"}],\"name\":\"getHistoricalConfigLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"governanceActionsConsumed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vaa\",\"type\":\"bytes\"}],\"name\":\"submitConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"configIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"threshold\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"}],\"name\":\"ChainConfigSet\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"digest\",\"type\":\"bytes32\"}],\"name\":\"GovernanceActionAlreadyConsumed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"action\",\"type\":\"uint8\"}],\"name\":\"InvalidAction\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"}],\"name\":\"InvalidChainId\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"}],\"name\":\"InvalidConfig\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"chainId\",\"type\":\"uint16\"}],\"name\":\"InvalidGovernanceChainId\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"contractAddress\",\"type\":\"bytes32\"}],\"name\":\"InvalidGovernanceContract\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"module\",\"type\":\"bytes32\"}],\"name\":\"InvalidModule\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nextConfigIndex\",\"type\":\"uint256\"}],\"name\":\"InvalidNextConfigIndex\",\"type\":\"error\"}]",
}

// DelegatedguardiansABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegatedguardiansMetaData.ABI instead.
var DelegatedguardiansABI = DelegatedguardiansMetaData.ABI

// Delegatedguardians is an auto generated Go binding around an Ethereum contract.
type Delegatedguardians struct {
	DelegatedguardiansCaller     // Read-only binding to the contract
	DelegatedguardiansTransactor // Write-only binding to the contract
	DelegatedguardiansFilterer   // Log filterer for contract events
}

// DelegatedguardiansCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegatedguardiansCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedguardiansTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegatedguardiansTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedguardiansFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegatedguardiansFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegatedguardiansSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegatedguardiansSession struct {
	Contract     *Delegatedguardians // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// DelegatedguardiansCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegatedguardiansCallerSession struct {
	Contract *DelegatedguardiansCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// DelegatedguardiansTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegatedguardiansTransactorSession struct {
	Contract     *DelegatedguardiansTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// DelegatedguardiansRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegatedguardiansRaw struct {
	Contract *Delegatedguardians // Generic contract binding to access the raw methods on
}

// DelegatedguardiansCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegatedguardiansCallerRaw struct {
	Contract *DelegatedguardiansCaller // Generic read-only contract binding to access the raw methods on
}

// DelegatedguardiansTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegatedguardiansTransactorRaw struct {
	Contract *DelegatedguardiansTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegatedguardians creates a new instance of Delegatedguardians, bound to a specific deployed contract.
func NewDelegatedguardians(address common.Address, backend bind.ContractBackend) (*Delegatedguardians, error) {
	contract, err := bindDelegatedguardians(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Delegatedguardians{DelegatedguardiansCaller: DelegatedguardiansCaller{contract: contract}, DelegatedguardiansTransactor: DelegatedguardiansTransactor{contract: contract}, DelegatedguardiansFilterer: DelegatedguardiansFilterer{contract: contract}}, nil
}

// NewDelegatedguardiansCaller creates a new read-only instance of Delegatedguardians, bound to a specific deployed contract.
func NewDelegatedguardiansCaller(address common.Address, caller bind.ContractCaller) (*DelegatedguardiansCaller, error) {
	contract, err := bindDelegatedguardians(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegatedguardiansCaller{contract: contract}, nil
}

// NewDelegatedguardiansTransactor creates a new write-only instance of Delegatedguardians, bound to a specific deployed contract.
func NewDelegatedguardiansTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegatedguardiansTransactor, error) {
	contract, err := bindDelegatedguardians(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegatedguardiansTransactor{contract: contract}, nil
}

// NewDelegatedguardiansFilterer creates a new log filterer instance of Delegatedguardians, bound to a specific deployed contract.
func NewDelegatedguardiansFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegatedguardiansFilterer, error) {
	contract, err := bindDelegatedguardians(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegatedguardiansFilterer{contract: contract}, nil
}

// bindDelegatedguardians binds a generic wrapper to an already deployed contract.
func bindDelegatedguardians(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DelegatedguardiansMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Delegatedguardians *DelegatedguardiansRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Delegatedguardians.Contract.DelegatedguardiansCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Delegatedguardians *DelegatedguardiansRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.DelegatedguardiansTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Delegatedguardians *DelegatedguardiansRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.DelegatedguardiansTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Delegatedguardians *DelegatedguardiansCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Delegatedguardians.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Delegatedguardians *DelegatedguardiansTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Delegatedguardians *DelegatedguardiansTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.contract.Transact(opts, method, params...)
}

// ChainIdsLength is a free data retrieval call binding the contract method 0x3e28ea8a.
//
// Solidity: function chainIdsLength() view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansCaller) ChainIdsLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "chainIdsLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChainIdsLength is a free data retrieval call binding the contract method 0x3e28ea8a.
//
// Solidity: function chainIdsLength() view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansSession) ChainIdsLength() (*big.Int, error) {
	return _Delegatedguardians.Contract.ChainIdsLength(&_Delegatedguardians.CallOpts)
}

// ChainIdsLength is a free data retrieval call binding the contract method 0x3e28ea8a.
//
// Solidity: function chainIdsLength() view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansCallerSession) ChainIdsLength() (*big.Int, error) {
	return _Delegatedguardians.Contract.ChainIdsLength(&_Delegatedguardians.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x01f6b5a2.
//
// Solidity: function getChainId(uint256 index) view returns(uint16)
func (_Delegatedguardians *DelegatedguardiansCaller) GetChainId(opts *bind.CallOpts, index *big.Int) (uint16, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getChainId", index)

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x01f6b5a2.
//
// Solidity: function getChainId(uint256 index) view returns(uint16)
func (_Delegatedguardians *DelegatedguardiansSession) GetChainId(index *big.Int) (uint16, error) {
	return _Delegatedguardians.Contract.GetChainId(&_Delegatedguardians.CallOpts, index)
}

// GetChainId is a free data retrieval call binding the contract method 0x01f6b5a2.
//
// Solidity: function getChainId(uint256 index) view returns(uint16)
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetChainId(index *big.Int) (uint16, error) {
	return _Delegatedguardians.Contract.GetChainId(&_Delegatedguardians.CallOpts, index)
}

// GetChainIds is a free data retrieval call binding the contract method 0x1d776323.
//
// Solidity: function getChainIds() view returns(uint16[])
func (_Delegatedguardians *DelegatedguardiansCaller) GetChainIds(opts *bind.CallOpts) ([]uint16, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getChainIds")

	if err != nil {
		return *new([]uint16), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint16)).(*[]uint16)

	return out0, err

}

// GetChainIds is a free data retrieval call binding the contract method 0x1d776323.
//
// Solidity: function getChainIds() view returns(uint16[])
func (_Delegatedguardians *DelegatedguardiansSession) GetChainIds() ([]uint16, error) {
	return _Delegatedguardians.Contract.GetChainIds(&_Delegatedguardians.CallOpts)
}

// GetChainIds is a free data retrieval call binding the contract method 0x1d776323.
//
// Solidity: function getChainIds() view returns(uint16[])
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetChainIds() ([]uint16, error) {
	return _Delegatedguardians.Contract.GetChainIds(&_Delegatedguardians.CallOpts)
}

// GetConfig is a free data retrieval call binding the contract method 0x70c852a1.
//
// Solidity: function getConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansCaller) GetConfig(opts *bind.CallOpts, _chainId uint16) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getConfig", _chainId)

	if err != nil {
		return *new(WormholeDelegatedGuardiansDelegatedGuardianSet), err
	}

	out0 := *abi.ConvertType(out[0], new(WormholeDelegatedGuardiansDelegatedGuardianSet)).(*WormholeDelegatedGuardiansDelegatedGuardianSet)

	return out0, err

}

// GetConfig is a free data retrieval call binding the contract method 0x70c852a1.
//
// Solidity: function getConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansSession) GetConfig(_chainId uint16) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetConfig(&_Delegatedguardians.CallOpts, _chainId)
}

// GetConfig is a free data retrieval call binding the contract method 0x70c852a1.
//
// Solidity: function getConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetConfig(_chainId uint16) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetConfig(&_Delegatedguardians.CallOpts, _chainId)
}

// GetConfig0 is a free data retrieval call binding the contract method 0xc3f909d4.
//
// Solidity: function getConfig() view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansCaller) GetConfig0(opts *bind.CallOpts) ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getConfig0")

	if err != nil {
		return *new([]WormholeDelegatedGuardiansDelegatedGuardianSet), err
	}

	out0 := *abi.ConvertType(out[0], new([]WormholeDelegatedGuardiansDelegatedGuardianSet)).(*[]WormholeDelegatedGuardiansDelegatedGuardianSet)

	return out0, err

}

// GetConfig0 is a free data retrieval call binding the contract method 0xc3f909d4.
//
// Solidity: function getConfig() view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansSession) GetConfig0() ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetConfig0(&_Delegatedguardians.CallOpts)
}

// GetConfig0 is a free data retrieval call binding the contract method 0xc3f909d4.
//
// Solidity: function getConfig() view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetConfig0() ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetConfig0(&_Delegatedguardians.CallOpts)
}

// GetHistoricalConfig is a free data retrieval call binding the contract method 0x5ac1b08b.
//
// Solidity: function getHistoricalConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansCaller) GetHistoricalConfig(opts *bind.CallOpts, _chainId uint16) ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getHistoricalConfig", _chainId)

	if err != nil {
		return *new([]WormholeDelegatedGuardiansDelegatedGuardianSet), err
	}

	out0 := *abi.ConvertType(out[0], new([]WormholeDelegatedGuardiansDelegatedGuardianSet)).(*[]WormholeDelegatedGuardiansDelegatedGuardianSet)

	return out0, err

}

// GetHistoricalConfig is a free data retrieval call binding the contract method 0x5ac1b08b.
//
// Solidity: function getHistoricalConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansSession) GetHistoricalConfig(_chainId uint16) ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfig(&_Delegatedguardians.CallOpts, _chainId)
}

// GetHistoricalConfig is a free data retrieval call binding the contract method 0x5ac1b08b.
//
// Solidity: function getHistoricalConfig(uint16 _chainId) view returns((uint16,uint32,uint8,address[])[])
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetHistoricalConfig(_chainId uint16) ([]WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfig(&_Delegatedguardians.CallOpts, _chainId)
}

// GetHistoricalConfig0 is a free data retrieval call binding the contract method 0x9f2c3a14.
//
// Solidity: function getHistoricalConfig(uint16 _chainId, uint256 _index) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansCaller) GetHistoricalConfig0(opts *bind.CallOpts, _chainId uint16, _index *big.Int) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getHistoricalConfig0", _chainId, _index)

	if err != nil {
		return *new(WormholeDelegatedGuardiansDelegatedGuardianSet), err
	}

	out0 := *abi.ConvertType(out[0], new(WormholeDelegatedGuardiansDelegatedGuardianSet)).(*WormholeDelegatedGuardiansDelegatedGuardianSet)

	return out0, err

}

// GetHistoricalConfig0 is a free data retrieval call binding the contract method 0x9f2c3a14.
//
// Solidity: function getHistoricalConfig(uint16 _chainId, uint256 _index) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansSession) GetHistoricalConfig0(_chainId uint16, _index *big.Int) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfig0(&_Delegatedguardians.CallOpts, _chainId, _index)
}

// GetHistoricalConfig0 is a free data retrieval call binding the contract method 0x9f2c3a14.
//
// Solidity: function getHistoricalConfig(uint16 _chainId, uint256 _index) view returns((uint16,uint32,uint8,address[]))
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetHistoricalConfig0(_chainId uint16, _index *big.Int) (WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfig0(&_Delegatedguardians.CallOpts, _chainId, _index)
}

// GetHistoricalConfigLength is a free data retrieval call binding the contract method 0xf00aa995.
//
// Solidity: function getHistoricalConfigLength(uint16 _chainId) view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansCaller) GetHistoricalConfigLength(opts *bind.CallOpts, _chainId uint16) (*big.Int, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "getHistoricalConfigLength", _chainId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetHistoricalConfigLength is a free data retrieval call binding the contract method 0xf00aa995.
//
// Solidity: function getHistoricalConfigLength(uint16 _chainId) view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansSession) GetHistoricalConfigLength(_chainId uint16) (*big.Int, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfigLength(&_Delegatedguardians.CallOpts, _chainId)
}

// GetHistoricalConfigLength is a free data retrieval call binding the contract method 0xf00aa995.
//
// Solidity: function getHistoricalConfigLength(uint16 _chainId) view returns(uint256)
func (_Delegatedguardians *DelegatedguardiansCallerSession) GetHistoricalConfigLength(_chainId uint16) (*big.Int, error) {
	return _Delegatedguardians.Contract.GetHistoricalConfigLength(&_Delegatedguardians.CallOpts, _chainId)
}

// GovernanceActionsConsumed is a free data retrieval call binding the contract method 0xa548ee64.
//
// Solidity: function governanceActionsConsumed(bytes32 ) view returns(bool)
func (_Delegatedguardians *DelegatedguardiansCaller) GovernanceActionsConsumed(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _Delegatedguardians.contract.Call(opts, &out, "governanceActionsConsumed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GovernanceActionsConsumed is a free data retrieval call binding the contract method 0xa548ee64.
//
// Solidity: function governanceActionsConsumed(bytes32 ) view returns(bool)
func (_Delegatedguardians *DelegatedguardiansSession) GovernanceActionsConsumed(arg0 [32]byte) (bool, error) {
	return _Delegatedguardians.Contract.GovernanceActionsConsumed(&_Delegatedguardians.CallOpts, arg0)
}

// GovernanceActionsConsumed is a free data retrieval call binding the contract method 0xa548ee64.
//
// Solidity: function governanceActionsConsumed(bytes32 ) view returns(bool)
func (_Delegatedguardians *DelegatedguardiansCallerSession) GovernanceActionsConsumed(arg0 [32]byte) (bool, error) {
	return _Delegatedguardians.Contract.GovernanceActionsConsumed(&_Delegatedguardians.CallOpts, arg0)
}

// SubmitConfig is a paid mutator transaction binding the contract method 0x555ff288.
//
// Solidity: function submitConfig(bytes vaa) returns()
func (_Delegatedguardians *DelegatedguardiansTransactor) SubmitConfig(opts *bind.TransactOpts, vaa []byte) (*types.Transaction, error) {
	return _Delegatedguardians.contract.Transact(opts, "submitConfig", vaa)
}

// SubmitConfig is a paid mutator transaction binding the contract method 0x555ff288.
//
// Solidity: function submitConfig(bytes vaa) returns()
func (_Delegatedguardians *DelegatedguardiansSession) SubmitConfig(vaa []byte) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.SubmitConfig(&_Delegatedguardians.TransactOpts, vaa)
}

// SubmitConfig is a paid mutator transaction binding the contract method 0x555ff288.
//
// Solidity: function submitConfig(bytes vaa) returns()
func (_Delegatedguardians *DelegatedguardiansTransactorSession) SubmitConfig(vaa []byte) (*types.Transaction, error) {
	return _Delegatedguardians.Contract.SubmitConfig(&_Delegatedguardians.TransactOpts, vaa)
}

// DelegatedguardiansChainConfigSetIterator is returned from FilterChainConfigSet and is used to iterate over the raw logs and unpacked data for ChainConfigSet events raised by the Delegatedguardians contract.
type DelegatedguardiansChainConfigSetIterator struct {
	Event *DelegatedguardiansChainConfigSet // Event containing the contract specifics and raw log

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
func (it *DelegatedguardiansChainConfigSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegatedguardiansChainConfigSet)
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
		it.Event = new(DelegatedguardiansChainConfigSet)
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
func (it *DelegatedguardiansChainConfigSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegatedguardiansChainConfigSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegatedguardiansChainConfigSet represents a ChainConfigSet event raised by the Delegatedguardians contract.
type DelegatedguardiansChainConfigSet struct {
	ConfigIndex *big.Int
	ChainId     uint16
	Threshold   uint8
	Keys        []common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChainConfigSet is a free log retrieval operation binding the contract event 0x60521c8f957274659667169ced5f227b61b730968b22ef66f02a72f27394eba4.
//
// Solidity: event ChainConfigSet(uint256 configIndex, uint16 chainId, uint8 threshold, address[] keys)
func (_Delegatedguardians *DelegatedguardiansFilterer) FilterChainConfigSet(opts *bind.FilterOpts) (*DelegatedguardiansChainConfigSetIterator, error) {

	logs, sub, err := _Delegatedguardians.contract.FilterLogs(opts, "ChainConfigSet")
	if err != nil {
		return nil, err
	}
	return &DelegatedguardiansChainConfigSetIterator{contract: _Delegatedguardians.contract, event: "ChainConfigSet", logs: logs, sub: sub}, nil
}

// WatchChainConfigSet is a free log subscription operation binding the contract event 0x60521c8f957274659667169ced5f227b61b730968b22ef66f02a72f27394eba4.
//
// Solidity: event ChainConfigSet(uint256 configIndex, uint16 chainId, uint8 threshold, address[] keys)
func (_Delegatedguardians *DelegatedguardiansFilterer) WatchChainConfigSet(opts *bind.WatchOpts, sink chan<- *DelegatedguardiansChainConfigSet) (event.Subscription, error) {

	logs, sub, err := _Delegatedguardians.contract.WatchLogs(opts, "ChainConfigSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegatedguardiansChainConfigSet)
				if err := _Delegatedguardians.contract.UnpackLog(event, "ChainConfigSet", log); err != nil {
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

// ParseChainConfigSet is a log parse operation binding the contract event 0x60521c8f957274659667169ced5f227b61b730968b22ef66f02a72f27394eba4.
//
// Solidity: event ChainConfigSet(uint256 configIndex, uint16 chainId, uint8 threshold, address[] keys)
func (_Delegatedguardians *DelegatedguardiansFilterer) ParseChainConfigSet(log types.Log) (*DelegatedguardiansChainConfigSet, error) {
	event := new(DelegatedguardiansChainConfigSet)
	if err := _Delegatedguardians.contract.UnpackLog(event, "ChainConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
