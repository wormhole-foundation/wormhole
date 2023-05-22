// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

// todo: move under libraries folder
import "../../interfaces/relayer/TypedUnits.sol";

error UnexpectedExecutionParamsVersion(uint8 version, uint8 expectedVersion);
error UnsupportedExecutionParamsVersion(uint8 version);
error TargetChainAndExecutionParamsVersionMismatch(uint16 targetChainId, uint8 version);
error UnexpectedQuoteParamsVersion(uint8 version, uint8 expectedVersion);
error UnsupportedQuoteParamsVersion(uint8 version);
error TargetChainAndQuoteParamsVersionMismatch(uint16 targetChainId, uint8 version);

enum ExecutionParamsVersion {
    EVM_V1
}

struct EvmExecutionParamsV1 {
    Gas gasLimit;
}

enum QuoteParamsVersion {
    EVM_V1
}

struct EvmQuoteParamsV1 {
    Wei targetChainRefundPerGasUsed;
}

function decodeExecutionParamsVersion(bytes memory data) pure returns (ExecutionParamsVersion version) {
    (version) = abi.decode(data, (ExecutionParamsVersion));
}

function encodeEvmExecutionParamsV1(EvmExecutionParamsV1 memory executionParams)
    pure
    returns (bytes memory)
{
    return abi.encode(
        ExecutionParamsVersion.EVM_V1, executionParams.gasLimit
    );
}

function decodeEvmExecutionParamsV1(bytes memory data)
    pure
    returns (EvmExecutionParamsV1 memory executionParams)
{
    uint8 version;
    (version, executionParams.gasLimit) =
        abi.decode(data, (uint8, Gas));

    if (version != uint8(ExecutionParamsVersion.EVM_V1)) 
        revert UnexpectedExecutionParamsVersion(version, uint8(ExecutionParamsVersion.EVM_V1));
}

function encodeEvmQuoteParamsV1(EvmQuoteParamsV1 memory quoteParams)
    pure
    returns (bytes memory)
{
    return abi.encode(
        ExecutionParamsVersion.EVM_V1, quoteParams.targetChainRefundPerGasUsed
    );
}

function decodeEvmQuoteParamsV1(bytes memory data)
    pure
    returns (EvmQuoteParamsV1 memory quoteParams)
{
    uint8 version;
    (version, quoteParams.targetChainRefundPerGasUsed) =
        abi.decode(data, (uint8, Wei));

    if (version != uint8(QuoteParamsVersion.EVM_V1)) 
        revert UnexpectedQuoteParamsVersion(version, uint8(QuoteParamsVersion.EVM_V1));
}

