// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";
import {BytesParsing} from "../../libraries/relayer/BytesParsing.sol";

error UnexpectedExecutionParamsVersion(uint8 version, uint8 expectedVersion);
error UnsupportedExecutionParamsVersion(uint8 version);
error TargetChainAndExecutionParamsVersionMismatch(uint16 targetChainId, uint8 version);
error UnexpectedExecutionInfoVersion(uint8 version, uint8 expectedVersion);
error UnsupportedExecutionInfoVersion(uint8 version);
error TargetChainAndExecutionInfoVersionMismatch(uint16 targetChainId, uint8 version);
error VersionMismatchOverride(uint8 instructionVersion, uint8 overrideVersion);
using BytesParsing for bytes;

enum ExecutionParamsVersion {
    EVM_V1
}

struct EvmExecutionParamsV1 {
    Gas gasLimit;
}

enum ExecutionInfoVersion {
    EVM_V1
}

struct EvmExecutionInfoV1 {
    Gas gasLimit;
    GasPrice targetChainRefundPerGasUnused;
}

function decodeExecutionParamsVersion(bytes memory data) pure returns (ExecutionParamsVersion version) {
    (uint8 _version,) = data.asUint8(0);
    version = ExecutionParamsVersion(_version);
}

function decodeExecutionInfoVersion(bytes memory data) pure returns (ExecutionInfoVersion version) {
    (uint8 _version,) = data.asUint8(0);
    version = ExecutionInfoVersion(_version);
}

function encodeEvmExecutionParamsV1(EvmExecutionParamsV1 memory executionParams)
    pure
    returns (bytes memory)
{
    return abi.encodePacked(
        uint8(ExecutionParamsVersion.EVM_V1), executionParams.gasLimit
    );
}

function decodeEvmExecutionParamsV1(bytes memory data)
    pure
    returns (EvmExecutionParamsV1 memory executionParams)
{
    (uint8 parsedVersion, uint offset) = data.asUint8(0);
    if(ExecutionParamsVersion(parsedVersion) != ExecutionParamsVersion.EVM_V1) {
        revert UnexpectedExecutionParamsVersion(parsedVersion, uint8(ExecutionParamsVersion.EVM_V1));
    }
    uint32 gasLimit;
    (gasLimit, offset) = data.asUint32(offset);
    executionParams.gasLimit = Gas.wrap(gasLimit);
}

function encodeEvmExecutionInfoV1(EvmExecutionInfoV1 memory executionInfo)
    pure
    returns (bytes memory)
{
    return abi.encodePacked(
        uint8(ExecutionInfoVersion.EVM_V1), executionInfo.gasLimit, executionInfo.targetChainRefundPerGasUnused
    );
}

function decodeEvmExecutionInfoV1(bytes memory data)
    pure
    returns (EvmExecutionInfoV1 memory executionInfo)
{

    (uint8 parsedVersion, uint offset) = data.asUint8(0);
    if(ExecutionInfoVersion(parsedVersion) != ExecutionInfoVersion.EVM_V1) {
        revert UnexpectedExecutionInfoVersion(parsedVersion, uint8(ExecutionInfoVersion.EVM_V1));
    }
    uint32 gasLimit;
    (gasLimit, offset) = data.asUint32(offset);
    executionInfo.gasLimit = Gas.wrap(gasLimit);
    uint256 targetChainRefundPerGasUnused;
    (targetChainRefundPerGasUnused, offset) = data.asUint256(offset);
    executionInfo.targetChainRefundPerGasUnused = GasPrice.wrap(targetChainRefundPerGasUnused);
}

function getEmptyEvmExecutionParamsV1() pure returns (EvmExecutionParamsV1 memory executionParams) {
    executionParams.gasLimit = Gas.wrap(uint256(0));
}

