// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

// todo: move under libraries folder
import "../../interfaces/relayer/TypedUnits.sol";

error UnexpectedExecutionParametersVersion(uint8 version, uint8 expectedVersion);

enum ExecutionParameterVersion {
    EVM_V1
}

struct EvmExecutionParametersV1 {
    Gas gasLimit;
    Wei targetChainRefundPerGasUsed;
}

function decodeExecutionParameterVersion(bytes memory data) pure returns (uint8 version) {
    (version) = abi.decode(data, (uint8));
}

function encodeEvmExecutionParametersV1(EvmExecutionParametersV1 memory executionParameters)
    pure
    returns (bytes memory)
{
    return abi.encode(
        ExecutionParameterVersion.EVM_V1, executionParameters.gasLimit, executionParameters.targetChainRefundPerGasUsed
    );
}

function decodeEvmExecutionParametersV1(bytes memory data)
    pure
    returns (EvmExecutionParametersV1 memory executionParameters)
{
    uint8 version;
    (version, executionParameters.gasLimit, executionParameters.targetChainRefundPerGasUsed) =
        abi.decode(data, (uint8, Gas, Wei));

    if (version != uint8(ExecutionParameterVersion.EVM_V1)) 
        revert UnexpectedExecutionParametersVersion(version, uint8(ExecutionParameterVersion.EVM_V1));
}
