// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormholeReceiver.sol";
import "../interfaces/IWormhole.sol";
import "../interfaces/IRelayProvider.sol";
import "../interfaces/IForwardInstructionViewer.sol";
import "../interfaces/IWormholeRelayerInternalStructs.sol";
import "../interfaces/IForwardWrapper.sol";
import "../interfaces/IWormholeReceiver.sol";
import "../interfaces/IRelayProvider.sol";

contract ForwardWrapper {
    IForwardInstructionViewer forwardInstructionViewer;
    IWormhole wormhole;

    error RequesterNotCoreRelayer();
    error ForwardNotSufficientlyFunded(uint256 amountOfFunds, uint256 amountOfFundsNeeded);

    constructor(address _wormholeRelayer, address _wormhole) {
        forwardInstructionViewer = IForwardInstructionViewer(_wormholeRelayer);
        wormhole = IWormhole(_wormhole);
    }

    function executeInstruction(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        IWormholeReceiver.DeliveryData memory data,
        bytes[] memory signedVaas
    ) public payable returns (bool callToTargetContractSucceeded, uint256 transactionFeeRefundAmount) {
        if (msg.sender != address(forwardInstructionViewer)) {
            revert RequesterNotCoreRelayer();
        }

        uint256 preGas = gasleft();

        // Calls the 'receiveWormholeMessages' endpoint on the contract 'instruction.targetAddress'
        // (with the gas limit and value specified in instruction, and 'encodedVMs' as the input)
        (callToTargetContractSucceeded,) = forwardInstructionViewer.fromWormholeFormat(instruction.targetAddress).call{
            gas: instruction.executionParameters.gasLimit,
            value: instruction.receiverValueTarget
        }(abi.encodeCall(IWormholeReceiver.receiveWormholeMessages, ((data), signedVaas)));

        uint256 postGas = gasleft();

        // Calculate the amount of gas used in the call (upperbounding at the gas limit, which shouldn't have been exceeded)
        uint256 gasUsed = (preGas - postGas) > instruction.executionParameters.gasLimit
            ? instruction.executionParameters.gasLimit
            : (preGas - postGas);

        // Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the fraction of gas unused)
        transactionFeeRefundAmount = (instruction.executionParameters.gasLimit - gasUsed)
            * instruction.maximumRefundTarget / instruction.executionParameters.gasLimit;

        IWormholeRelayerInternalStructs.ForwardInstruction memory forwardInstruction =
            forwardInstructionViewer.getForwardInstruction();

        if (forwardInstruction.isValid) {
            uint256 feeForForward = transactionFeeRefundAmount + forwardInstruction.msgValue;
            if (feeForForward < forwardInstruction.totalFee) {
                revert ForwardNotSufficientlyFunded(feeForForward, forwardInstruction.totalFee);
            }
        }

        if (!callToTargetContractSucceeded) {
            msg.sender.call{value: msg.value}("");
        }
    }

    function safeRelayProviderSupportsChain(IRelayProvider relayProvider, uint16 chainId) view external returns (bool isSupported){
        return relayProvider.isChainSupported(chainId);
    }
}
