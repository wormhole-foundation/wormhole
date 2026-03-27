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
     * @notice Upgrades the Wormhole core bridge implementation contract via a governance VAA.
     * @dev Upgrades a contract via Governance VAA/VM
     *      This function is called with a VAA produced by the guardian network on the governance
     *      chain (Solana). The VAA's payload encodes the new implementation address.
     *      Reverts if called on a forked chain — use `submitRecoverChainId` first on forks.
     * @param _vm The raw binary governance VAA authorizing the upgrade.
     */
    function submitContractUpgrade(
        bytes memory _vm
    ) public {
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
     * @notice Updates the message publishing fee via a governance VAA.
     * @dev Sets a `messageFee` via Governance VAA/VM
     *      The new fee is encoded in the VAA payload. After this call, `publishMessage`
     *      callers must send exactly the new fee in wei or the call reverts.
     *      Reverts on forked chains (to prevent replay of fee-change VAAs from the original chain).
     * @param _vm The raw binary governance VAA authorizing the fee change.
     */
    function submitSetMessageFee(
        bytes memory _vm
    ) public {
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
     * @notice Rotates the active guardian set via a governance VAA.
     * @dev Deploys a new `guardianSet` via Governance VAA/VM
     *      The new guardian set's index must be exactly one greater than the current index.
     *      The old guardian set is given a 24-hour expiry window to allow in-flight VAAs to settle.
     *      The new guardian set must be non-empty to guard against accidental lockout.
     *      Reverts on forked chains unless `upgrade.chain == 0` (chain-agnostic upgrade).
     * @param _vm The raw binary governance VAA encoding the new guardian set.
     */
    function submitNewGuardianSet(
        bytes memory _vm
    ) public {
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
        require(
            upgrade.newGuardianSetIndex == getCurrentGuardianSetIndex() + 1,
            "index must increase in steps of 1"
        );

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
     * @notice Transfers accumulated message fees to a recipient address via a governance VAA.
     * @dev Submits transfer fees to the recipient via Governance VAA/VM
     *      Message fees accumulate in the contract as `msg.value` from `publishMessage` calls.
     *      The governance VAA encodes the amount and recipient address (as a 32-byte left-padded value).
     *      Reverts on forked chains unless `transfer.chain == 0` (chain-agnostic transfer).
     * @param _vm The raw binary governance VAA authorizing the fee transfer.
     */
    function submitTransferFees(
        bytes memory _vm
    ) public {
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
     * @notice Re-synchronizes the Wormhole chain ID and EVM chain ID after a hard fork.
     * @dev Updates the `chainId` and `evmChainId` on a forked chain via Governance VAA/VM
     *      This function is only callable when `isFork()` returns true (i.e. `block.chainid` differs
     *      from the stored `evmChainId`). It allows a forked chain to adopt a new Wormhole chain ID
     *      so governance and upgrades can resume normally without replaying the original chain's VAAs.
     * @param _vm The raw binary governance VAA encoding the new chain IDs.
     */
    function submitRecoverChainId(
        bytes memory _vm
    ) public {
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
     * @notice Replaces the current implementation contract with a new one and calls its `initialize()`.
     * @dev Upgrades the `currentImplementation` with a `newImplementation`
     *      Uses ERC-1967 proxy storage to update the implementation slot. After the upgrade,
     *      `initialize()` is called via `delegatecall` to set up the new implementation's state.
     *      Emits `ContractUpgraded` with the old and new implementation addresses.
     * @param newImplementation The address of the new implementation contract.
     */
    function upgradeImplementation(
        address newImplementation
    ) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) =
            newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        require(success, string(reason));

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    /**
     * @dev Verifies a Governance VAA/VM is valid
     *      Checks that the VAA: is signed by the current guardian set, originates from the
     *      governance chain (Solana), is emitted by the governance contract, and has not already
     *      been consumed (replay protection).
     */
    function verifyGovernanceVM(
        Structs.VM memory vm
    ) internal view returns (bool, string memory) {
        // Verify the VAA is valid
        (bool isValid, string memory reason) = verifyVM(vm);
        if (!isValid) {
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
        if (governanceActionIsConsumed(vm.hash)) {
            return (false, "governance action already consumed");
        }

        // Confirm the governance VAA/VM is valid
        return (true, "");
    }
}
