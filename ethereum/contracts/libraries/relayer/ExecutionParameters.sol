// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

// todo: move under libraries folder
import "../../interfaces/relayer/TypedUnits.sol";

struct EvmExecutionParamtersV1 {
    Gas gasLimit;
    Wei targetChainRefundPerGasUsed;
}

function encodeEvmExecutionParamtersV1(EvmExecutionParamtersV1 memory executionParameters)
    pure
    returns (bytes memory)
{
    return abi.encode(executionParameters.gasLimit, executionParameters.targetChainRefundPerGasUsed);
}

function decodeEvmExecutionParametersV1(bytes memory data)
    pure
    returns (EvmExecutionParamtersV1 memory executionParameters)
{
    (executionParameters.gasLimit, executionParameters.targetChainRefundPerGasUsed) =
        abi.decode(data, (Gas, Wei));
}
