// The Custom Consistency Level feature allows integrators to specify custom finality handling for their observations.
// It involves reading the custom configuration from an on chain contract, using the emitter address as a key. If an
// entry is found, then the specified special handling is performed. Currently, the only supported custom handling is
// to wait a certain number of blocks after the block containing the message reaches the specified finality
// (finalized, safe or immediate).
//
// To generate the ABI bindings for the CustomConsistencyLevel contract do the following:
//
// - Install abigen: go install github.com/ethereum/go-ethereum/cmd/abigen@latest
// - Copy the ABI definitions from ethereum/build-forge/CustomConsistencyLevel.sol/CustomConsistencyLevel.json (the stuff after `"abi":`) into /tmp/CustomConsistencyLevel.abi.
// - cd node/pkg/watcher/evm
// - mkdir custom_consistency_level_abi
// - abigen --abi /tmp/CustomConsistencyLevel.abi --pkg ccl --out custom_consistency_level_abi/CustomConsistencyLevel.go

package evm

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	cclAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/custom_consistency_level_abi"

	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CCLRequestType used to represent the custom handling type.
type CCLRequestType uint8

const (
	NothingSpecialType CCLRequestType = iota
	AdditionalBlocksType
)

type (
	// CCLRequest is the standard interface for custom request types.
	CCLRequest interface {
		Type() CCLRequestType
	}

	// NothingSpecial means there is no custom handling enabled for this emitter.
	NothingSpecial struct{}

	// AdditionalBlocks means this emitter is configured for the additional blocks custom handling.
	AdditionalBlocks struct {
		consistencyLevel uint8
		additionalBlocks uint16
	}
)

func (nr *NothingSpecial) Type() CCLRequestType {
	return NothingSpecialType
}

func (abr *AdditionalBlocks) Type() CCLRequestType {
	return AdditionalBlocksType
}

type CCLMap map[vaa.ChainID]string

var (
	// cclMainnetMap specifies the custom consistency level contracts for each mainnet chain.
	cclMainnetMap = CCLMap{}

	// cclTestnetMap specifies the custom consistency level contracts for each testnet chain.
	cclTestnetMap = CCLMap{}

	// cclDevnetMap specifies the custom consistency level contracts for each devnet chain.
	cclDevnetMap = CCLMap{
		vaa.ChainIDEthereum: "0x6A4B4A882F5F0a447078b4Fd0b4B571A82371ec2",
	}

	// cclEmptyData is used to check for an empty response from the contract, meaning the emitter address is not configured for special handling.
	cclEmptyData [32]byte
)

type (
	// CCLCacheEntry is the data stored in the cache for a given emitter
	CCLCacheEntry struct {
		data     [32]byte
		readTime time.Time
	}

	// CCLCache is the layout of the cache of config data for emitters
	CCLCache map[ethCommon.Address]CCLCacheEntry
)

// CCLCacheTimeoutInterval is the lifetime of a cache entry. After that time, we delete the entry and re-read the config data.
// TODO: This time is arbitrary. Does it sound okay?
const CCLCacheTimeoutInterval = time.Minute * 5

// cclEnable enables the custom consistency level feature, if it is configured for this environment / chain.
func (w *Watcher) cclEnable(ctx context.Context) error {
	addrStr, err := cclGetContractAddr(w.env, w.chainID)
	if err != nil {
		return err
	}

	if addrStr == "" {
		// This is not an error. It just means the feature is not enabled for this chain.
		w.cclEnabled = false
		return nil
	}

	w.cclAddr = ethCommon.HexToAddress(addrStr)
	w.cclLogger.Info("custom consistency level is enabled", zap.Stringer("contractAddr", w.cclAddr))

	// Do a test read on the contract to confirm it exists. This should not return anything, but it shouldn't fail!
	// We use the free function here so we don't add the zero emitter to the cache.
	_, err = CCLReadContract(ctx, w.ethConn.Client(), w.cclAddr, ethCommon.Address{})
	if err != nil {
		w.cclLogger.Error("failed to do test read on contract, disabling custom consistency level handling", zap.Stringer("contractAddr", w.cclAddr), zap.Error(err))
		return nil
	}

	w.cclEnabled = true
	w.cclCacheLock.Lock()
	w.cclCache = CCLCache{}
	w.cclCacheLock.Unlock()
	return nil
}

// cclHandleMessage is called for new observations that have the consistency level set to custom handling.
// It reads the configuration for the emitter and updates the `pendingMessage` object for custom handling.
func (w *Watcher) cclHandleMessage(parentCtx context.Context, pe *pendingMessage, emitterAddr ethCommon.Address) {
	if !w.cclEnabled {
		w.cclLogger.Error("received an observation with custom handling but the feature is not enabled, treating as finalized", zap.String("msgId", pe.message.MessageIDString()))
		pe.message.ConsistencyLevel = vaa.ConsistencyLevelFinalized
		return
	}

	if pe.message.ConsistencyLevel != vaa.ConsistencyLevelCustom {
		w.cclLogger.Error("cclHandleMessage called with an invalid consistency level, ignoring it!",
			zap.String("msgId", pe.message.MessageIDString()),
			zap.Uint8("consistencyLevel", pe.message.ConsistencyLevel),
		)
		return
	}

	r, err := w.cclReadAndParseConfig(parentCtx, emitterAddr)
	if err != nil {
		w.cclLogger.Error("failed to look up config for custom handling, treating as finalized", zap.String("msgId", pe.message.MessageIDString()), zap.Error(err))
		pe.message.ConsistencyLevel = vaa.ConsistencyLevelFinalized
		return
	}

	switch req := r.(type) {
	case *NothingSpecial:
		w.cclLogger.Info("received an observation with the nothing special specifier, treating as finalized", zap.String("msgId", pe.message.MessageIDString()))
		pe.message.ConsistencyLevel = vaa.ConsistencyLevelFinalized
	case *AdditionalBlocks:
		if req.consistencyLevel != vaa.ConsistencyLevelFinalized && req.consistencyLevel != vaa.ConsistencyLevelSafe && req.consistencyLevel != vaa.ConsistencyLevelPublishImmediately {
			w.cclLogger.Error("received an observation with an additional blocks specifier but the configured consistency level is invalid, treating as finalized",
				zap.String("msgId", pe.message.MessageIDString()),
				zap.Uint8("consistencyLevel", req.consistencyLevel),
				zap.Uint16("additionalBlocks", req.additionalBlocks),
			)
			pe.message.ConsistencyLevel = vaa.ConsistencyLevelFinalized
			return
		}

		w.cclLogger.Info("received an observation with an additional blocks specifier",
			zap.String("msgId", pe.message.MessageIDString()),
			zap.Uint8("consistencyLevel", req.consistencyLevel),
			zap.Uint16("additionalBlocks", req.additionalBlocks),
		)
		pe.message.ConsistencyLevel = req.consistencyLevel
		pe.additionalBlocks = uint64(req.additionalBlocks)
	default:
		w.cclLogger.Error("invalid custom handling type, treating as finalized", zap.Stringer("emitterAddress", emitterAddr), zap.Uint8("reqType", uint8(req.Type())), zap.Error(err))
		pe.message.ConsistencyLevel = vaa.ConsistencyLevelFinalized
	}
}

// cclReadAndParseConfig reads the configuration for a given emitter and parses it into a request type.
func (w *Watcher) cclReadAndParseConfig(ctx context.Context, emitterAddr ethCommon.Address) (CCLRequest, error) {
	data, err := w.cclReadContract(ctx, emitterAddr)
	if err != nil {
		return &NothingSpecial{}, err
	}

	if data == cclEmptyData {
		return &NothingSpecial{}, nil
	}

	request, err := cclParseConfig(data)
	if err != nil {
		return &NothingSpecial{}, fmt.Errorf("failed to parse contract data: %w", err)
	}

	return request, err
}

// cclReadContract calls into the contract to read the configuration for a given emitter.
func (w *Watcher) cclReadContract(ctx context.Context, emitterAddr ethCommon.Address) ([32]byte, error) {
	// Before we read the config from the contract, see if we already have it in the cache.
	data, found := w.cclCacheLookUp(emitterAddr)
	if found {
		return data, nil
	}

	data, err := CCLReadContract(ctx, w.ethConn.Client(), w.cclAddr, emitterAddr)
	if err != nil {
		return cclEmptyData, err
	}

	w.cclCacheUpdate(emitterAddr, data)
	w.cclLogger.Debug("read contract", zap.Stringer("emitterAddr", emitterAddr))
	return data, nil
}

// CCLReadContract calls into the contract to read the configuration for a given emitter.
// This is a free function so it can be called by the config verification tool.
func CCLReadContract(ctx context.Context, ethClient *ethclient.Client, cclAddr ethCommon.Address, emitterAddr ethCommon.Address) ([32]byte, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	caller, err := cclAbi.NewCclCaller(cclAddr, ethClient)
	if err != nil {
		return cclEmptyData, fmt.Errorf("failed to create abi caller: %w", err)
	}

	data, err := caller.GetConfiguration(&ethBind.CallOpts{Context: timeout}, emitterAddr)
	if err != nil {
		return cclEmptyData, err
	}

	return data, nil
}

// cclParseConfig parses the configuration data returned by the contract into a request.
func cclParseConfig(data [32]byte) (CCLRequest, error) {
	reader := bytes.NewReader(data[:])

	t := CCLRequestType(0)
	if err := binary.Read(reader, binary.BigEndian, &t); err != nil {
		return nil, fmt.Errorf("failed to read data type: %w", err)
	}

	if t == 0x01 {
		return cclParseAdditionalBlocksConfig(reader)
	}

	if t == 0x00 {
		return &NothingSpecial{}, nil
	}

	return nil, fmt.Errorf("unexpected data type: %d", t)
}

// cclParseAdditionalBlocksConfig parses the configuration for an additional blocks request.
// Note that the configuration type (the first byte) has already been read and verified.
func cclParseAdditionalBlocksConfig(reader *bytes.Reader) (CCLRequest, error) {
	consistencyLevel := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &consistencyLevel); err != nil {
		return nil, fmt.Errorf("failed to read consistency level: %w", err)
	}

	blocks := uint16(0)
	if err := binary.Read(reader, binary.BigEndian, &blocks); err != nil {
		return nil, fmt.Errorf("failed to read num blocks: %w", err)
	}

	// The config data is 32 bytes and the AdditionalBlocks request uses the first four, so we should have 28 left.
	if reader.Len() != 28 {
		return nil, fmt.Errorf("unexpected remaining unread bytes in buffer, should be 28, are %d", reader.Len())
	}

	return &AdditionalBlocks{consistencyLevel, blocks}, nil
}

// cclGetContractAddr returns the contract address for the given environment / chain.
// If the chain is not configured to use custom consistency level handling, the empty string is returned.
func cclGetContractAddr(env common.Environment, chainID vaa.ChainID) (string, error) {
	m, err := cclGetContractAddrMap(env)
	if err != nil {
		return "", err
	}

	addrStr, exists := m[chainID]
	if !exists {
		// The entry not existing is not an error. It just means we don't support custom consistency levels on this chain.
		return "", nil
	}

	return addrStr, nil
}

// cclGetContractAddrMap returns the configuration map for the given environment.
func cclGetContractAddrMap(env common.Environment) (CCLMap, error) {
	if env == common.MainNet {
		return cclMainnetMap, nil
	}

	if env == common.TestNet {
		return cclTestnetMap, nil
	}

	if env == common.UnsafeDevNet {
		return cclDevnetMap, nil
	}

	return CCLMap{}, ErrInvalidEnv
}

// cclCacheLookUp looks to see if the configuration for an emitter is currently in our cache.
// If the entry does not exists, we return "not found". Otherwise, if it is not expired, we return it.
// If it is expired, we delete it from the cache and return "not found".
func (w *Watcher) cclCacheLookUp(emitterAddr ethCommon.Address) ([32]byte, bool) {
	w.cclCacheLock.Lock()
	defer w.cclCacheLock.Unlock()

	if entry, exists := w.cclCache[emitterAddr]; exists {
		if time.Since(entry.readTime) < CCLCacheTimeoutInterval {
			return entry.data, true
		}

		w.cclLogger.Debug("cache entry has expired", zap.Stringer("emitterAddr", emitterAddr))
		delete(w.cclCache, emitterAddr)
	}

	return cclEmptyData, false
}

// cclCacheUpdate updates the entry in the cache for a given emitter, including the read time.
func (w *Watcher) cclCacheUpdate(emitterAddr ethCommon.Address, data [32]byte) {
	w.cclCacheLock.Lock()
	w.cclCache[emitterAddr] = CCLCacheEntry{data, time.Now()}
	w.cclCacheLock.Unlock()
	w.cclLogger.Debug("cache entry added", zap.Stringer("emitterAddr", emitterAddr))
}
