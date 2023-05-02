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

    function getValuesFromRelayProvider(address providerAddress, uint16 targetChain, uint256 receiverValue)
        external
        view
        returns (address rewardAddress, uint256 maximumBudget, uint256 receiverValueTarget);
}
