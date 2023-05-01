// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./IWormholeRelayerInternalStructs.sol";
import "./IWormholeReceiver.sol";
import "./IRelayProvider.sol";

interface IForwardWrapper {
    function executeInstruction(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        IWormholeReceiver.DeliveryData memory deliveryData,
        bytes[] memory signedVaas
    ) external payable returns (bool callToTargetContractSucceeded, uint256 transactionFeeRefundAmount);

    function safeRelayProviderSupportsChain(IRelayProvider relayProvider, uint16 chainId)
        external
        view
        returns (bool isSupported);
}
