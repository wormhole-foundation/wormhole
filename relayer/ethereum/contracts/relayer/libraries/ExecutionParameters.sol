// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";
import {BytesParsing} from "../../relayer/libraries/BytesParsing.sol";

error UnexpectedExecutionParamsVersion(uint8 version, uint8 expectedVersion);
error UnsupportedExecutionParamsVersion(uint8 version);
error TargetChainAndExecutionParamsVersionMismatch(uint16 targetChain, uint8 version);
error UnexpectedExecutionInfoVersion(uint8 version, uint8 expectedVersion);
error UnsupportedExecutionInfoVersion(uint8 version);
error TargetChainAndExecutionInfoVersionMismatch(uint16 targetChain, uint8 version);
error VersionMismatchOverride(uint8 instructionVersion, uint8 overrideVersion);

using BytesParsing for bytes;

enum ExecutionParamsVersion {EVM_V1}

struct EvmExecutionParamsV1 {
    Gas gasLimit;
}

enum ExecutionInfoVersion {EVM_V1}

struct EvmExecutionInfoV1 {
    Gas gasLimit;
    GasPrice targetChainRefundPerGasUnused;
}

function decodeExecutionParamsVersion(bytes memory data)
    pure
    returns (ExecutionParamsVersion version)
{
    (version) = abi.decode(data, (ExecutionParamsVersion));
}

function decodeExecutionInfoVersion(bytes memory data)
    pure
    returns (ExecutionInfoVersion version)
{
    (version) = abi.decode(data, (ExecutionInfoVersion));
}

function encodeEvmExecutionParamsV1(EvmExecutionParamsV1 memory executionParams)
    pure
    returns (bytes memory)
{
    return abi.encode(uint8(ExecutionParamsVersion.EVM_V1), executionParams.gasLimit);
}

function decodeEvmExecutionParamsV1(bytes memory data)
    pure
    returns (EvmExecutionParamsV1 memory executionParams)
{
    uint8 version;
    (version, executionParams.gasLimit) = abi.decode(data, (uint8, Gas));

    if (version != uint8(ExecutionParamsVersion.EVM_V1)) {
        revert UnexpectedExecutionParamsVersion(version, uint8(ExecutionParamsVersion.EVM_V1));
    }
}

function encodeEvmExecutionInfoV1(EvmExecutionInfoV1 memory executionInfo)
    pure
    returns (bytes memory)
{
    return abi.encode(
        uint8(ExecutionInfoVersion.EVM_V1),
        executionInfo.gasLimit,
        executionInfo.targetChainRefundPerGasUnused
    );
}

function decodeEvmExecutionInfoV1(bytes memory data)
    pure
    returns (EvmExecutionInfoV1 memory executionInfo)
{
    uint8 version;
    (version, executionInfo.gasLimit, executionInfo.targetChainRefundPerGasUnused) =
        abi.decode(data, (uint8, Gas, GasPrice));

    if (version != uint8(ExecutionInfoVersion.EVM_V1)) {
        revert UnexpectedExecutionInfoVersion(version, uint8(ExecutionInfoVersion.EVM_V1));
    }
}

function getEmptyEvmExecutionParamsV1()
    pure
    returns (EvmExecutionParamsV1 memory executionParams)
{
    executionParams.gasLimit = Gas.wrap(uint256(0));
}

