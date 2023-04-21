// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/IWormholeReceiver.sol";
import "../../interfaces/IWormhole.sol";
import "../../interfaces/relayer/IRelayProvider.sol";
import "../../interfaces/relayer/IForwardInstructionViewer.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import "../../interfaces/relayer/IForwardWrapper.sol";
import "../../interfaces/relayer/IWormholeReceiver.sol";
import "../../interfaces/relayer/IRelayProvider.sol";
import {CoreRelayerLibrary} from "../coreRelayer/CoreRelayerLibrary.sol";

contract ForwardWrapper is CoreRelayerLibrary {
    IForwardInstructionViewer public forwardInstructionViewer;
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
        }(abi.encodeWithSelector(IWormholeReceiver.receiveWormholeMessages.selector, data, signedVaas));

        uint256 postGas = gasleft();

        // Calculate the amount of gas used in the call (upperbounding at the gas limit, which shouldn't have been exceeded)
        uint256 gasUsed = (preGas - postGas) > instruction.executionParameters.gasLimit
            ? instruction.executionParameters.gasLimit
            : (preGas - postGas);

        // Calculate the amount of maxTransactionFee to refund (multiply the maximum refund by the fraction of gas unused)
        transactionFeeRefundAmount = (instruction.executionParameters.gasLimit - gasUsed)
            * instruction.maximumRefundTarget / instruction.executionParameters.gasLimit;

        IWormholeRelayerInternalStructs.ForwardInstruction[] memory forwardInstructions =
            forwardInstructionViewer.getForwardInstructions();

        if (forwardInstructions.length > 0) {
            uint256 totalMsgValue = 0;
            uint256 totalFee = 0;
            for (uint8 i = 0; i < forwardInstructions.length; i++) {
                totalMsgValue += forwardInstructions[i].msgValue;
                totalFee += forwardInstructions[i].totalFee;
            }
            uint256 feeForForward = transactionFeeRefundAmount + totalMsgValue;
            if (feeForForward < totalFee) {
                revert ForwardNotSufficientlyFunded(feeForForward, totalFee);
            }
        }

        if (!callToTargetContractSucceeded) {
            msg.sender.call{value: msg.value}("");
        }
    }

    function safeRelayProviderSupportsChain(IRelayProvider relayProvider, uint16 chainId)
        external
        view
        returns (bool isSupported)
    {
        return relayProvider.isChainSupported(chainId);
    }

    error InvalidFork();
    error InvalidGovernanceVM(string reason);

    function submitContractUpgrade(bytes memory _vm) external {
        if (isFork()) {
            revert InvalidFork();
        }

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(_vm);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        CoreRelayerLibrary.ContractUpgrade memory contractUpgrade = CoreRelayerLibrary.parseUpgrade(vm.payload, module);
        if (contractUpgrade.chain != chainId()) {
            revert WrongChainId(contractUpgrade.chain);
        }

        upgradeImplementation(contractUpgrade.newContract);
    }

    function evmChainId() public view returns (uint256) {
        return _state.evmChainId;
    }

    function isFork() public view returns (bool) {
        return evmChainId() != block.chainid;
    }

    function verifyGovernanceVM(bytes memory encodedVM)
        internal
        view
        returns (IWormhole.VM memory parsedVM, bool isValid, string memory invalidReason)
    {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVM);

        if (!valid) {
            return (vm, valid, reason);
        }

        if (vm.emitterChainId != governanceChainId()) {
            return (vm, false, "wrong governance chain");
        }
        if (vm.emitterAddress != governanceContract()) {
            return (vm, false, "wrong governance contract");
        }

        if (governanceActionIsConsumed(vm.hash)) {
            return (vm, false, "governance action already consumed");
        }

        return (vm, true, "");
    }

    function setConsumedGovernanceAction(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }
}
