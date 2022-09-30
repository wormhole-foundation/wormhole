// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi_arbitrum

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

// AbiArbitrumABI is the input ABI used to generate the binding from.
const AbiArbitrumABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"deposit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"estimateRetryableTicket\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"leaf\",\"type\":\"uint64\"}],\"name\":\"constructOutboxProof\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"send\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"blockNum\",\"type\":\"uint64\"}],\"name\":\"findBatchContainingBlock\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"batch\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"getL1Confirmations\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"confirmations\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"contractCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"gasEstimateComponents\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"gasEstimate\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"gasEstimateForL1\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFeeEstimate\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"contractCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"gasEstimateL1Component\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"gasEstimateForL1\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFeeEstimate\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchNum\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"index\",\"type\":\"uint64\"}],\"name\":\"legacyLookupMessageBatchProof\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"path\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1Dest\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"calldataForL1\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nitroGenesisBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]"

// AbiArbitrum is an auto generated Go binding around an Ethereum contract.
type AbiArbitrum struct {
	AbiArbitrumCaller     // Read-only binding to the contract
	AbiArbitrumTransactor // Write-only binding to the contract
	AbiArbitrumFilterer   // Log filterer for contract events
}

// AbiArbitrumCaller is an auto generated read-only Go binding around an Ethereum contract.
type AbiArbitrumCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiArbitrumTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AbiArbitrumTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiArbitrumFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AbiArbitrumFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbiArbitrumSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AbiArbitrumSession struct {
	Contract     *AbiArbitrum      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AbiArbitrumCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AbiArbitrumCallerSession struct {
	Contract *AbiArbitrumCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// AbiArbitrumTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AbiArbitrumTransactorSession struct {
	Contract     *AbiArbitrumTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// AbiArbitrumRaw is an auto generated low-level Go binding around an Ethereum contract.
type AbiArbitrumRaw struct {
	Contract *AbiArbitrum // Generic contract binding to access the raw methods on
}

// AbiArbitrumCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AbiArbitrumCallerRaw struct {
	Contract *AbiArbitrumCaller // Generic read-only contract binding to access the raw methods on
}

// AbiArbitrumTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AbiArbitrumTransactorRaw struct {
	Contract *AbiArbitrumTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAbiArbitrum creates a new instance of AbiArbitrum, bound to a specific deployed contract.
func NewAbiArbitrum(address common.Address, backend bind.ContractBackend) (*AbiArbitrum, error) {
	contract, err := bindAbiArbitrum(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AbiArbitrum{AbiArbitrumCaller: AbiArbitrumCaller{contract: contract}, AbiArbitrumTransactor: AbiArbitrumTransactor{contract: contract}, AbiArbitrumFilterer: AbiArbitrumFilterer{contract: contract}}, nil
}

// NewAbiArbitrumCaller creates a new read-only instance of AbiArbitrum, bound to a specific deployed contract.
func NewAbiArbitrumCaller(address common.Address, caller bind.ContractCaller) (*AbiArbitrumCaller, error) {
	contract, err := bindAbiArbitrum(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AbiArbitrumCaller{contract: contract}, nil
}

// NewAbiArbitrumTransactor creates a new write-only instance of AbiArbitrum, bound to a specific deployed contract.
func NewAbiArbitrumTransactor(address common.Address, transactor bind.ContractTransactor) (*AbiArbitrumTransactor, error) {
	contract, err := bindAbiArbitrum(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AbiArbitrumTransactor{contract: contract}, nil
}

// NewAbiArbitrumFilterer creates a new log filterer instance of AbiArbitrum, bound to a specific deployed contract.
func NewAbiArbitrumFilterer(address common.Address, filterer bind.ContractFilterer) (*AbiArbitrumFilterer, error) {
	contract, err := bindAbiArbitrum(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AbiArbitrumFilterer{contract: contract}, nil
}

// bindAbiArbitrum binds a generic wrapper to an already deployed contract.
func bindAbiArbitrum(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AbiArbitrumABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbiArbitrum *AbiArbitrumRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbiArbitrum.Contract.AbiArbitrumCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbiArbitrum *AbiArbitrumRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.AbiArbitrumTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbiArbitrum *AbiArbitrumRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.AbiArbitrumTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbiArbitrum *AbiArbitrumCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbiArbitrum.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbiArbitrum *AbiArbitrumTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbiArbitrum *AbiArbitrumTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.contract.Transact(opts, method, params...)
}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_AbiArbitrum *AbiArbitrumCaller) ConstructOutboxProof(opts *bind.CallOpts, size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	var out []interface{}
	err := _AbiArbitrum.contract.Call(opts, &out, "constructOutboxProof", size, leaf)

	outstruct := new(struct {
		Send  [32]byte
		Root  [32]byte
		Proof [][32]byte
	})

	outstruct.Send = out[0].([32]byte)
	outstruct.Root = out[1].([32]byte)
	outstruct.Proof = out[2].([][32]byte)

	return *outstruct, err

}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_AbiArbitrum *AbiArbitrumSession) ConstructOutboxProof(size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	return _AbiArbitrum.Contract.ConstructOutboxProof(&_AbiArbitrum.CallOpts, size, leaf)
}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_AbiArbitrum *AbiArbitrumCallerSession) ConstructOutboxProof(size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	return _AbiArbitrum.Contract.ConstructOutboxProof(&_AbiArbitrum.CallOpts, size, leaf)
}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_AbiArbitrum *AbiArbitrumCaller) FindBatchContainingBlock(opts *bind.CallOpts, blockNum uint64) (uint64, error) {
	var out []interface{}
	err := _AbiArbitrum.contract.Call(opts, &out, "findBatchContainingBlock", blockNum)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_AbiArbitrum *AbiArbitrumSession) FindBatchContainingBlock(blockNum uint64) (uint64, error) {
	return _AbiArbitrum.Contract.FindBatchContainingBlock(&_AbiArbitrum.CallOpts, blockNum)
}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_AbiArbitrum *AbiArbitrumCallerSession) FindBatchContainingBlock(blockNum uint64) (uint64, error) {
	return _AbiArbitrum.Contract.FindBatchContainingBlock(&_AbiArbitrum.CallOpts, blockNum)
}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_AbiArbitrum *AbiArbitrumCaller) GetL1Confirmations(opts *bind.CallOpts, blockHash [32]byte) (uint64, error) {
	var out []interface{}
	err := _AbiArbitrum.contract.Call(opts, &out, "getL1Confirmations", blockHash)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_AbiArbitrum *AbiArbitrumSession) GetL1Confirmations(blockHash [32]byte) (uint64, error) {
	return _AbiArbitrum.Contract.GetL1Confirmations(&_AbiArbitrum.CallOpts, blockHash)
}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_AbiArbitrum *AbiArbitrumCallerSession) GetL1Confirmations(blockHash [32]byte) (uint64, error) {
	return _AbiArbitrum.Contract.GetL1Confirmations(&_AbiArbitrum.CallOpts, blockHash)
}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_AbiArbitrum *AbiArbitrumCaller) LegacyLookupMessageBatchProof(opts *bind.CallOpts, batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	var out []interface{}
	err := _AbiArbitrum.contract.Call(opts, &out, "legacyLookupMessageBatchProof", batchNum, index)

	outstruct := new(struct {
		Proof         [][32]byte
		Path          *big.Int
		L2Sender      common.Address
		L1Dest        common.Address
		L2Block       *big.Int
		L1Block       *big.Int
		Timestamp     *big.Int
		Amount        *big.Int
		CalldataForL1 []byte
	})

	outstruct.Proof = out[0].([][32]byte)
	outstruct.Path = out[1].(*big.Int)
	outstruct.L2Sender = out[2].(common.Address)
	outstruct.L1Dest = out[3].(common.Address)
	outstruct.L2Block = out[4].(*big.Int)
	outstruct.L1Block = out[5].(*big.Int)
	outstruct.Timestamp = out[6].(*big.Int)
	outstruct.Amount = out[7].(*big.Int)
	outstruct.CalldataForL1 = out[8].([]byte)

	return *outstruct, err

}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_AbiArbitrum *AbiArbitrumSession) LegacyLookupMessageBatchProof(batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	return _AbiArbitrum.Contract.LegacyLookupMessageBatchProof(&_AbiArbitrum.CallOpts, batchNum, index)
}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_AbiArbitrum *AbiArbitrumCallerSession) LegacyLookupMessageBatchProof(batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	return _AbiArbitrum.Contract.LegacyLookupMessageBatchProof(&_AbiArbitrum.CallOpts, batchNum, index)
}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_AbiArbitrum *AbiArbitrumCaller) NitroGenesisBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AbiArbitrum.contract.Call(opts, &out, "nitroGenesisBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_AbiArbitrum *AbiArbitrumSession) NitroGenesisBlock() (*big.Int, error) {
	return _AbiArbitrum.Contract.NitroGenesisBlock(&_AbiArbitrum.CallOpts)
}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_AbiArbitrum *AbiArbitrumCallerSession) NitroGenesisBlock() (*big.Int, error) {
	return _AbiArbitrum.Contract.NitroGenesisBlock(&_AbiArbitrum.CallOpts)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_AbiArbitrum *AbiArbitrumTransactor) EstimateRetryableTicket(opts *bind.TransactOpts, sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.contract.Transact(opts, "estimateRetryableTicket", sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_AbiArbitrum *AbiArbitrumSession) EstimateRetryableTicket(sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.EstimateRetryableTicket(&_AbiArbitrum.TransactOpts, sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_AbiArbitrum *AbiArbitrumTransactorSession) EstimateRetryableTicket(sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.EstimateRetryableTicket(&_AbiArbitrum.TransactOpts, sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumTransactor) GasEstimateComponents(opts *bind.TransactOpts, to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.contract.Transact(opts, "gasEstimateComponents", to, contractCreation, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumSession) GasEstimateComponents(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.GasEstimateComponents(&_AbiArbitrum.TransactOpts, to, contractCreation, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumTransactorSession) GasEstimateComponents(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.GasEstimateComponents(&_AbiArbitrum.TransactOpts, to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumTransactor) GasEstimateL1Component(opts *bind.TransactOpts, to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.contract.Transact(opts, "gasEstimateL1Component", to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumSession) GasEstimateL1Component(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.GasEstimateL1Component(&_AbiArbitrum.TransactOpts, to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_AbiArbitrum *AbiArbitrumTransactorSession) GasEstimateL1Component(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _AbiArbitrum.Contract.GasEstimateL1Component(&_AbiArbitrum.TransactOpts, to, contractCreation, data)
}
