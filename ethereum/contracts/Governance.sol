// contracts/Governance.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./Structs.sol";
import "./GovernanceStructs.sol";
import "./Messages.sol";
import "./Setters.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
 * @dev `Governance` defines a means to enacting changes to the core bridge contract,
 * guardianSets, message fees, and transfer fees
 */
abstract contract Governance is GovernanceStructs, Messages, Setters, ERC1967Upgrade {
    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    event GuardianSetAdded(uint32 indexed index);

    // "Core" (left padded)
    bytes32 constant module = 0x00000000000000000000000000000000000000000000000000000000436f7265;

    /**
     * @dev Upgrades a contract via Governance VAA/VM
     */
    function submitContractUpgrade(bytes memory _vm) public {
        require(!isFork(), "invalid fork");

        Structs.VM memory vm = parseVM(_vm);

        // Verify the VAA is valid before processing it
        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        GovernanceStructs.ContractUpgrade memory upgrade = parseContractUpgrade(vm.payload);

        // Verify the VAA is for this module
        require(upgrade.module == module, "Invalid Module");

        // Verify the VAA is for this chain
        require(upgrade.chain == chainId(), "Invalid Chain");

        // Record the governance action as consumed
        setGovernanceActionConsumed(vm.hash);

        // Upgrades the implementation to the new contract
        upgradeImplementation(upgrade.newContract);
    }

    /**
     * @dev Sets a `messageFee` via Governance VAA/VM
     */
    function submitSetMessageFee(bytes memory _vm) public {
        Structs.VM memory vm = parseVM(_vm);

        // Verify the VAA is valid before processing it
        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        GovernanceStructs.SetMessageFee memory upgrade = parseSetMessageFee(vm.payload);

        // Verify the VAA is for this module
        require(upgrade.module == module, "Invalid Module");

        // Verify the VAA is for this chain
        require(upgrade.chain == chainId() && !isFork(), "Invalid Chain");

        // Record the governance action as consumed to prevent reentry
        setGovernanceActionConsumed(vm.hash);

        // Updates the messageFee
        setMessageFee(upgrade.messageFee);
    }

    /**
     * @dev Deploys a new `guardianSet` via Governance VAA/VM
     */
    function submitNewGuardianSet(bytes memory _vm) public {
        Structs.VM memory vm = parseVM(_vm);

        // Verify the VAA is valid before processing it
        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        GovernanceStructs.GuardianSetUpgrade memory upgrade = parseGuardianSetUpgrade(vm.payload);

        // Verify the VAA is for this module
        require(upgrade.module == module, "invalid Module");

        // Verify the VAA is for this chain
        require((upgrade.chain == chainId() && !isFork()) || upgrade.chain == 0, "invalid Chain");

        // Verify the Guardian Set keys are not empty, this guards
        // against the accidential upgrade to an empty GuardianSet
        require(upgrade.newGuardianSet.keys.length > 0, "new guardian set is empty");

        // Verify that the index is incrementing via a predictable +1 pattern
        require(upgrade.newGuardianSetIndex == getCurrentGuardianSetIndex() + 1, "index must increase in steps of 1");

        // Record the governance action as consumed to prevent reentry
        setGovernanceActionConsumed(vm.hash);

        // Trigger a time-based expiry of current guardianSet
        expireGuardianSet(getCurrentGuardianSetIndex());

        // Add the new guardianSet to guardianSets
        storeGuardianSet(upgrade.newGuardianSet, upgrade.newGuardianSetIndex);

        // Makes the new guardianSet effective
        updateGuardianSetIndex(upgrade.newGuardianSetIndex);
    }

    /**
     * @dev Submits transfer fees to the recipient via Governance VAA/VM
     */
    function submitTransferFees(bytes memory _vm) public {
        Structs.VM memory vm = parseVM(_vm);

        // Verify the VAA is valid before processing it
        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        // Obtains the transfer from the VAA payload
        GovernanceStructs.TransferFees memory transfer = parseTransferFees(vm.payload);

        // Verify the VAA is for this module
        require(transfer.module == module, "invalid Module");

        // Verify the VAA is for this chain
        require((transfer.chain == chainId() && !isFork()) || transfer.chain == 0, "invalid Chain");

        // Record the governance action as consumed to prevent reentry
        setGovernanceActionConsumed(vm.hash);

        // Obtains the recipient address to be paid transfer fees
        address payable recipient = payable(address(uint160(uint256(transfer.recipient))));

        // Transfers transfer fees to the recipient
        recipient.transfer(transfer.amount);
    }

    /**
    * @dev Updates the `chainId` and `evmChainId` on a forked chain via Governance VAA/VM
    */
    function submitRecoverChainId(bytes memory _vm) public {
        require(isFork(), "not a fork");

        Structs.VM memory vm = parseVM(_vm);

        // Verify the VAA is valid before processing it
        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        GovernanceStructs.RecoverChainId memory rci = parseRecoverChainId(vm.payload);

        // Verify the VAA is for this module
        require(rci.module == module, "invalid Module");

        // Verify the VAA is for this chain
        require(rci.evmChainId == block.chainid, "invalid EVM Chain");

        // Record the governance action as consumed to prevent reentry
        setGovernanceActionConsumed(vm.hash);

        // Update the chainIds
        setEvmChainId(rci.evmChainId);
        setChainId(rci.newChainId);
    }

    /**
     * @dev Upgrades the `currentImplementation` with a `newImplementation`
     */
    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        require(success, string(reason));

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    /**
     * @dev Verifies a Governance VAA/VM is valid
     */
    function verifyGovernanceVM(Structs.VM memory vm) internal view returns (bool, string memory){
        // Verify the VAA is valid
        (bool isValid, string memory reason) = verifyVM(vm);
        if (!isValid){
            return (false, reason);
        }

        // only current guardianset can sign governance packets
        if (vm.guardianSetIndex != getCurrentGuardianSetIndex()) {
            return (false, "not signed by current guardian set");
        }

        // Verify the VAA is from the governance chain (Solana)
        if (uint16(vm.emitterChainId) != governanceChainId()) {
            return (false, "wrong governance chain");
        }

        // Verify the emitter contract is the governance contract (0x4 left padded)
        if (vm.emitterAddress != governanceContract()) {
            return (false, "wrong governance contract");
        }

        // Verify this governance action hasn't already been
        // consumed to prevent reentry and replay
        if (governanceActionIsConsumed(vm.hash)){
            return (false, "governance action already consumed");
        }

        // Confirm the governance VAA/VM is valid
        return (true, "");
    }
}