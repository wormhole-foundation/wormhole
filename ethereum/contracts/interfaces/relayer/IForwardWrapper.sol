// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormholeRelayerInternalStructs.sol";
import "../interfaces/IWormholeReceiver.sol";
import "../interfaces/IRelayProvider.sol";

interface IForwardWrapper {
    function executeInstruction(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        IWormholeReceiver.DeliveryData memory deliveryData,
        bytes[] memory signedVaas
    ) external payable returns (bool callToTargetContractSucceeded, uint256 transactionFeeRefundAmount);

    function safeRelayProviderSupportsChain(IRelayProvider relayProvider) external view returns(bool isSupported);
}
