// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package optimism_ctc_abi

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

// Lib_OVMCodecQueueElement is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecQueueElement struct {
	TransactionHash [32]byte
	Timestamp       *big.Int
	BlockNumber     *big.Int
}

// OptimismCtcAbiABI is the input ABI used to generate the binding from.
const OptimismCtcAbiABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxTransactionGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueGasCost\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueL2GasPrepaid\",\"type\":\"uint256\"}],\"name\":\"L2GasParamsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"QueueBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"SequencerBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"TransactionBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l1TxOrigin\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"TransactionEnqueued\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_ROLLUP_TX_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_ROLLUP_TX_GAS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"appendSequencerBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractIChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"enqueue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueGasCost\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueL2GasPrepaid\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockNumber\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastTimestamp\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextQueueIndex\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumPendingQueueElements\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getQueueElement\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"transactionHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint40\",\"name\":\"timestamp\",\"type\":\"uint40\"},{\"internalType\":\"uint40\",\"name\":\"blockNumber\",\"type\":\"uint40\"}],\"internalType\":\"structLib_OVMCodec.QueueElement\",\"name\":\"_element\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getQueueLength\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2GasDiscountDivisor\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTransactionGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"name\":\"setGasParams\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// OptimismCtcAbi is an auto generated Go binding around an Ethereum contract.
type OptimismCtcAbi struct {
	OptimismCtcAbiCaller     // Read-only binding to the contract
	OptimismCtcAbiTransactor // Write-only binding to the contract
	OptimismCtcAbiFilterer   // Log filterer for contract events
}

// OptimismCtcAbiCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismCtcAbiCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismCtcAbiTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismCtcAbiTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismCtcAbiFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismCtcAbiFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismCtcAbiSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismCtcAbiSession struct {
	Contract     *OptimismCtcAbi   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismCtcAbiCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismCtcAbiCallerSession struct {
	Contract *OptimismCtcAbiCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OptimismCtcAbiTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismCtcAbiTransactorSession struct {
	Contract     *OptimismCtcAbiTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OptimismCtcAbiRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismCtcAbiRaw struct {
	Contract *OptimismCtcAbi // Generic contract binding to access the raw methods on
}

// OptimismCtcAbiCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismCtcAbiCallerRaw struct {
	Contract *OptimismCtcAbiCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismCtcAbiTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismCtcAbiTransactorRaw struct {
	Contract *OptimismCtcAbiTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismCtcAbi creates a new instance of OptimismCtcAbi, bound to a specific deployed contract.
func NewOptimismCtcAbi(address common.Address, backend bind.ContractBackend) (*OptimismCtcAbi, error) {
	contract, err := bindOptimismCtcAbi(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbi{OptimismCtcAbiCaller: OptimismCtcAbiCaller{contract: contract}, OptimismCtcAbiTransactor: OptimismCtcAbiTransactor{contract: contract}, OptimismCtcAbiFilterer: OptimismCtcAbiFilterer{contract: contract}}, nil
}

// NewOptimismCtcAbiCaller creates a new read-only instance of OptimismCtcAbi, bound to a specific deployed contract.
func NewOptimismCtcAbiCaller(address common.Address, caller bind.ContractCaller) (*OptimismCtcAbiCaller, error) {
	contract, err := bindOptimismCtcAbi(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiCaller{contract: contract}, nil
}

// NewOptimismCtcAbiTransactor creates a new write-only instance of OptimismCtcAbi, bound to a specific deployed contract.
func NewOptimismCtcAbiTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismCtcAbiTransactor, error) {
	contract, err := bindOptimismCtcAbi(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiTransactor{contract: contract}, nil
}

// NewOptimismCtcAbiFilterer creates a new log filterer instance of OptimismCtcAbi, bound to a specific deployed contract.
func NewOptimismCtcAbiFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismCtcAbiFilterer, error) {
	contract, err := bindOptimismCtcAbi(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiFilterer{contract: contract}, nil
}

// bindOptimismCtcAbi binds a generic wrapper to an already deployed contract.
func bindOptimismCtcAbi(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismCtcAbiABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismCtcAbi *OptimismCtcAbiRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismCtcAbi.Contract.OptimismCtcAbiCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismCtcAbi *OptimismCtcAbiRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.OptimismCtcAbiTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismCtcAbi *OptimismCtcAbiRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.OptimismCtcAbiTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismCtcAbi *OptimismCtcAbiCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismCtcAbi.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismCtcAbi *OptimismCtcAbiTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismCtcAbi *OptimismCtcAbiTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.contract.Transact(opts, method, params...)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) MAXROLLUPTXSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "MAX_ROLLUP_TX_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MAXROLLUPTXSIZE(&_OptimismCtcAbi.CallOpts)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MAXROLLUPTXSIZE(&_OptimismCtcAbi.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) MINROLLUPTXGAS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "MIN_ROLLUP_TX_GAS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MINROLLUPTXGAS(&_OptimismCtcAbi.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MINROLLUPTXGAS(&_OptimismCtcAbi.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "batches")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiSession) Batches() (common.Address, error) {
	return _OptimismCtcAbi.Contract.Batches(&_OptimismCtcAbi.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) Batches() (common.Address, error) {
	return _OptimismCtcAbi.Contract.Batches(&_OptimismCtcAbi.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) EnqueueGasCost(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "enqueueGasCost")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) EnqueueGasCost() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.EnqueueGasCost(&_OptimismCtcAbi.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) EnqueueGasCost() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.EnqueueGasCost(&_OptimismCtcAbi.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) EnqueueL2GasPrepaid(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "enqueueL2GasPrepaid")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.EnqueueL2GasPrepaid(&_OptimismCtcAbi.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.EnqueueL2GasPrepaid(&_OptimismCtcAbi.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetLastBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getLastBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetLastBlockNumber() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetLastBlockNumber(&_OptimismCtcAbi.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetLastBlockNumber() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetLastBlockNumber(&_OptimismCtcAbi.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetLastTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getLastTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetLastTimestamp() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetLastTimestamp(&_OptimismCtcAbi.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetLastTimestamp() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetLastTimestamp(&_OptimismCtcAbi.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetNextQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getNextQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetNextQueueIndex() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetNextQueueIndex(&_OptimismCtcAbi.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetNextQueueIndex() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetNextQueueIndex(&_OptimismCtcAbi.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetNumPendingQueueElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getNumPendingQueueElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetNumPendingQueueElements(&_OptimismCtcAbi.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetNumPendingQueueElements(&_OptimismCtcAbi.CallOpts)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetQueueElement(opts *bind.CallOpts, _index *big.Int) (Lib_OVMCodecQueueElement, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getQueueElement", _index)

	if err != nil {
		return *new(Lib_OVMCodecQueueElement), err
	}

	out0 := *abi.ConvertType(out[0], new(Lib_OVMCodecQueueElement)).(*Lib_OVMCodecQueueElement)

	return out0, err

}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _OptimismCtcAbi.Contract.GetQueueElement(&_OptimismCtcAbi.CallOpts, _index)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _OptimismCtcAbi.Contract.GetQueueElement(&_OptimismCtcAbi.CallOpts, _index)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetQueueLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getQueueLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetQueueLength() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetQueueLength(&_OptimismCtcAbi.CallOpts)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetQueueLength() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetQueueLength(&_OptimismCtcAbi.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getTotalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetTotalBatches() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetTotalBatches(&_OptimismCtcAbi.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetTotalBatches() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetTotalBatches(&_OptimismCtcAbi.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "getTotalElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiSession) GetTotalElements() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetTotalElements(&_OptimismCtcAbi.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) GetTotalElements() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.GetTotalElements(&_OptimismCtcAbi.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) L2GasDiscountDivisor(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "l2GasDiscountDivisor")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.L2GasDiscountDivisor(&_OptimismCtcAbi.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.L2GasDiscountDivisor(&_OptimismCtcAbi.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiSession) LibAddressManager() (common.Address, error) {
	return _OptimismCtcAbi.Contract.LibAddressManager(&_OptimismCtcAbi.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) LibAddressManager() (common.Address, error) {
	return _OptimismCtcAbi.Contract.LibAddressManager(&_OptimismCtcAbi.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) MaxTransactionGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "maxTransactionGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MaxTransactionGasLimit(&_OptimismCtcAbi.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _OptimismCtcAbi.Contract.MaxTransactionGasLimit(&_OptimismCtcAbi.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _OptimismCtcAbi.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiSession) Resolve(_name string) (common.Address, error) {
	return _OptimismCtcAbi.Contract.Resolve(&_OptimismCtcAbi.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OptimismCtcAbi *OptimismCtcAbiCallerSession) Resolve(_name string) (common.Address, error) {
	return _OptimismCtcAbi.Contract.Resolve(&_OptimismCtcAbi.CallOpts, _name)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactor) AppendSequencerBatch(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismCtcAbi.contract.Transact(opts, "appendSequencerBatch")
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OptimismCtcAbi *OptimismCtcAbiSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.AppendSequencerBatch(&_OptimismCtcAbi.TransactOpts)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactorSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.AppendSequencerBatch(&_OptimismCtcAbi.TransactOpts)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactor) Enqueue(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OptimismCtcAbi.contract.Transact(opts, "enqueue", _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OptimismCtcAbi *OptimismCtcAbiSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.Enqueue(&_OptimismCtcAbi.TransactOpts, _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactorSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.Enqueue(&_OptimismCtcAbi.TransactOpts, _target, _gasLimit, _data)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactor) SetGasParams(opts *bind.TransactOpts, _l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _OptimismCtcAbi.contract.Transact(opts, "setGasParams", _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_OptimismCtcAbi *OptimismCtcAbiSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.SetGasParams(&_OptimismCtcAbi.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_OptimismCtcAbi *OptimismCtcAbiTransactorSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _OptimismCtcAbi.Contract.SetGasParams(&_OptimismCtcAbi.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// OptimismCtcAbiL2GasParamsUpdatedIterator is returned from FilterL2GasParamsUpdated and is used to iterate over the raw logs and unpacked data for L2GasParamsUpdated events raised by the OptimismCtcAbi contract.
type OptimismCtcAbiL2GasParamsUpdatedIterator struct {
	Event *OptimismCtcAbiL2GasParamsUpdated // Event containing the contract specifics and raw log

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
func (it *OptimismCtcAbiL2GasParamsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismCtcAbiL2GasParamsUpdated)
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
		it.Event = new(OptimismCtcAbiL2GasParamsUpdated)
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
func (it *OptimismCtcAbiL2GasParamsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismCtcAbiL2GasParamsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismCtcAbiL2GasParamsUpdated represents a L2GasParamsUpdated event raised by the OptimismCtcAbi contract.
type OptimismCtcAbiL2GasParamsUpdated struct {
	L2GasDiscountDivisor *big.Int
	EnqueueGasCost       *big.Int
	EnqueueL2GasPrepaid  *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterL2GasParamsUpdated is a free log retrieval operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) FilterL2GasParamsUpdated(opts *bind.FilterOpts) (*OptimismCtcAbiL2GasParamsUpdatedIterator, error) {

	logs, sub, err := _OptimismCtcAbi.contract.FilterLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiL2GasParamsUpdatedIterator{contract: _OptimismCtcAbi.contract, event: "L2GasParamsUpdated", logs: logs, sub: sub}, nil
}

// WatchL2GasParamsUpdated is a free log subscription operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) WatchL2GasParamsUpdated(opts *bind.WatchOpts, sink chan<- *OptimismCtcAbiL2GasParamsUpdated) (event.Subscription, error) {

	logs, sub, err := _OptimismCtcAbi.contract.WatchLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismCtcAbiL2GasParamsUpdated)
				if err := _OptimismCtcAbi.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
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

// ParseL2GasParamsUpdated is a log parse operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) ParseL2GasParamsUpdated(log types.Log) (*OptimismCtcAbiL2GasParamsUpdated, error) {
	event := new(OptimismCtcAbiL2GasParamsUpdated)
	if err := _OptimismCtcAbi.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OptimismCtcAbiQueueBatchAppendedIterator is returned from FilterQueueBatchAppended and is used to iterate over the raw logs and unpacked data for QueueBatchAppended events raised by the OptimismCtcAbi contract.
type OptimismCtcAbiQueueBatchAppendedIterator struct {
	Event *OptimismCtcAbiQueueBatchAppended // Event containing the contract specifics and raw log

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
func (it *OptimismCtcAbiQueueBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismCtcAbiQueueBatchAppended)
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
		it.Event = new(OptimismCtcAbiQueueBatchAppended)
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
func (it *OptimismCtcAbiQueueBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismCtcAbiQueueBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismCtcAbiQueueBatchAppended represents a QueueBatchAppended event raised by the OptimismCtcAbi contract.
type OptimismCtcAbiQueueBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterQueueBatchAppended is a free log retrieval operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) FilterQueueBatchAppended(opts *bind.FilterOpts) (*OptimismCtcAbiQueueBatchAppendedIterator, error) {

	logs, sub, err := _OptimismCtcAbi.contract.FilterLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiQueueBatchAppendedIterator{contract: _OptimismCtcAbi.contract, event: "QueueBatchAppended", logs: logs, sub: sub}, nil
}

// WatchQueueBatchAppended is a free log subscription operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) WatchQueueBatchAppended(opts *bind.WatchOpts, sink chan<- *OptimismCtcAbiQueueBatchAppended) (event.Subscription, error) {

	logs, sub, err := _OptimismCtcAbi.contract.WatchLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismCtcAbiQueueBatchAppended)
				if err := _OptimismCtcAbi.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
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

// ParseQueueBatchAppended is a log parse operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) ParseQueueBatchAppended(log types.Log) (*OptimismCtcAbiQueueBatchAppended, error) {
	event := new(OptimismCtcAbiQueueBatchAppended)
	if err := _OptimismCtcAbi.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OptimismCtcAbiSequencerBatchAppendedIterator is returned from FilterSequencerBatchAppended and is used to iterate over the raw logs and unpacked data for SequencerBatchAppended events raised by the OptimismCtcAbi contract.
type OptimismCtcAbiSequencerBatchAppendedIterator struct {
	Event *OptimismCtcAbiSequencerBatchAppended // Event containing the contract specifics and raw log

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
func (it *OptimismCtcAbiSequencerBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismCtcAbiSequencerBatchAppended)
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
		it.Event = new(OptimismCtcAbiSequencerBatchAppended)
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
func (it *OptimismCtcAbiSequencerBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismCtcAbiSequencerBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismCtcAbiSequencerBatchAppended represents a SequencerBatchAppended event raised by the OptimismCtcAbi contract.
type OptimismCtcAbiSequencerBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchAppended is a free log retrieval operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) FilterSequencerBatchAppended(opts *bind.FilterOpts) (*OptimismCtcAbiSequencerBatchAppendedIterator, error) {

	logs, sub, err := _OptimismCtcAbi.contract.FilterLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiSequencerBatchAppendedIterator{contract: _OptimismCtcAbi.contract, event: "SequencerBatchAppended", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchAppended is a free log subscription operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) WatchSequencerBatchAppended(opts *bind.WatchOpts, sink chan<- *OptimismCtcAbiSequencerBatchAppended) (event.Subscription, error) {

	logs, sub, err := _OptimismCtcAbi.contract.WatchLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismCtcAbiSequencerBatchAppended)
				if err := _OptimismCtcAbi.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
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

// ParseSequencerBatchAppended is a log parse operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) ParseSequencerBatchAppended(log types.Log) (*OptimismCtcAbiSequencerBatchAppended, error) {
	event := new(OptimismCtcAbiSequencerBatchAppended)
	if err := _OptimismCtcAbi.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OptimismCtcAbiTransactionBatchAppendedIterator is returned from FilterTransactionBatchAppended and is used to iterate over the raw logs and unpacked data for TransactionBatchAppended events raised by the OptimismCtcAbi contract.
type OptimismCtcAbiTransactionBatchAppendedIterator struct {
	Event *OptimismCtcAbiTransactionBatchAppended // Event containing the contract specifics and raw log

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
func (it *OptimismCtcAbiTransactionBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismCtcAbiTransactionBatchAppended)
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
		it.Event = new(OptimismCtcAbiTransactionBatchAppended)
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
func (it *OptimismCtcAbiTransactionBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismCtcAbiTransactionBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismCtcAbiTransactionBatchAppended represents a TransactionBatchAppended event raised by the OptimismCtcAbi contract.
type OptimismCtcAbiTransactionBatchAppended struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterTransactionBatchAppended is a free log retrieval operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) FilterTransactionBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OptimismCtcAbiTransactionBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OptimismCtcAbi.contract.FilterLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiTransactionBatchAppendedIterator{contract: _OptimismCtcAbi.contract, event: "TransactionBatchAppended", logs: logs, sub: sub}, nil
}

// WatchTransactionBatchAppended is a free log subscription operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) WatchTransactionBatchAppended(opts *bind.WatchOpts, sink chan<- *OptimismCtcAbiTransactionBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OptimismCtcAbi.contract.WatchLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismCtcAbiTransactionBatchAppended)
				if err := _OptimismCtcAbi.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
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

// ParseTransactionBatchAppended is a log parse operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) ParseTransactionBatchAppended(log types.Log) (*OptimismCtcAbiTransactionBatchAppended, error) {
	event := new(OptimismCtcAbiTransactionBatchAppended)
	if err := _OptimismCtcAbi.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OptimismCtcAbiTransactionEnqueuedIterator is returned from FilterTransactionEnqueued and is used to iterate over the raw logs and unpacked data for TransactionEnqueued events raised by the OptimismCtcAbi contract.
type OptimismCtcAbiTransactionEnqueuedIterator struct {
	Event *OptimismCtcAbiTransactionEnqueued // Event containing the contract specifics and raw log

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
func (it *OptimismCtcAbiTransactionEnqueuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismCtcAbiTransactionEnqueued)
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
		it.Event = new(OptimismCtcAbiTransactionEnqueued)
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
func (it *OptimismCtcAbiTransactionEnqueuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismCtcAbiTransactionEnqueuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismCtcAbiTransactionEnqueued represents a TransactionEnqueued event raised by the OptimismCtcAbi contract.
type OptimismCtcAbiTransactionEnqueued struct {
	L1TxOrigin common.Address
	Target     common.Address
	GasLimit   *big.Int
	Data       []byte
	QueueIndex *big.Int
	Timestamp  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionEnqueued is a free log retrieval operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) FilterTransactionEnqueued(opts *bind.FilterOpts, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (*OptimismCtcAbiTransactionEnqueuedIterator, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _OptimismCtcAbi.contract.FilterLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return &OptimismCtcAbiTransactionEnqueuedIterator{contract: _OptimismCtcAbi.contract, event: "TransactionEnqueued", logs: logs, sub: sub}, nil
}

// WatchTransactionEnqueued is a free log subscription operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) WatchTransactionEnqueued(opts *bind.WatchOpts, sink chan<- *OptimismCtcAbiTransactionEnqueued, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (event.Subscription, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _OptimismCtcAbi.contract.WatchLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismCtcAbiTransactionEnqueued)
				if err := _OptimismCtcAbi.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
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

// ParseTransactionEnqueued is a log parse operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_OptimismCtcAbi *OptimismCtcAbiFilterer) ParseTransactionEnqueued(log types.Log) (*OptimismCtcAbiTransactionEnqueued, error) {
	event := new(OptimismCtcAbiTransactionEnqueued)
	if err := _OptimismCtcAbi.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
		return nil, err
	}
	return event, nil
}
