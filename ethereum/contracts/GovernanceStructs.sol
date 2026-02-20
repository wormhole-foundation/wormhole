// contracts/GovernanceStructs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./libraries/external/BytesLib.sol";
import "./Structs.sol";

/**
 * @dev `GovernanceStructs` defines a set of structs and parsing functions
 * for minimal struct validation
 */
contract GovernanceStructs {
    using BytesLib for bytes;

    /// @notice Enumerates the supported governance action types.
    /// @dev Used internally and by off-chain tooling. The numeric action codes in the VAA
    ///      payload (1–5) correspond to the governance functions on the core bridge.
    enum GovernanceAction {
        UpgradeContract,
        UpgradeGuardianset
    }

    /// @notice Payload for governance action 1: upgrade the core bridge implementation contract.
    struct ContractUpgrade {
        /// @notice Identifies the Wormhole sub-module this governance action targets.
        /// @dev Must equal the "Core" module identifier (bytes32("Core") left-padded to 32 bytes).
        bytes32 module;
        /// @notice The governance action type code. Must be 1 for a contract upgrade.
        uint8 action;
        /// @notice The Wormhole chain ID this upgrade applies to.
        /// @dev Only the chain matching this ID should execute the upgrade.
        uint16 chain;
        /// @notice The address of the new implementation contract to upgrade to.
        address newContract;
    }

    /// @notice Payload for governance action 2: rotate the active guardian set.
    struct GuardianSetUpgrade {
        /// @notice Identifies the Wormhole sub-module this governance action targets.
        /// @dev Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice The governance action type code. Must be 2 for a guardian set upgrade.
        uint8 action;
        /// @notice The Wormhole chain ID this guardian set upgrade applies to (0 = all chains).
        uint16 chain;
        /// @notice The new guardian set to install, containing the new set of guardian public key addresses.
        Structs.GuardianSet newGuardianSet;
        /// @notice The version index of the new guardian set.
        /// @dev Must be exactly `getCurrentGuardianSetIndex() + 1` to enforce sequential upgrades.
        uint32 newGuardianSetIndex;
    }

    /// @notice Payload for governance action 3: update the message publishing fee.
    struct SetMessageFee {
        /// @notice Identifies the Wormhole sub-module this governance action targets.
        /// @dev Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice The governance action type code. Must be 3 for a fee update.
        uint8 action;
        /// @notice The Wormhole chain ID this fee change applies to.
        uint16 chain;
        /// @notice The new message fee in wei that callers must pay when calling `publishMessage`.
        uint256 messageFee;
    }

    /// @notice Payload for governance action 4: transfer accumulated message fees to a recipient.
    struct TransferFees {
        /// @notice Identifies the Wormhole sub-module this governance action targets.
        /// @dev Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice The governance action type code. Must be 4 for a fee transfer.
        uint8 action;
        /// @notice The Wormhole chain ID this transfer applies to (0 = all chains).
        uint16 chain;
        /// @notice The amount of wei to transfer to the recipient.
        uint256 amount;
        /// @notice The recipient address, left-padded to 32 bytes (EVM address in the last 20 bytes).
        bytes32 recipient;
    }

    /// @notice Payload for governance action 5: recover a hard-forked chain's chain IDs.
    /// @dev Used to resynchronize a forked chain's Wormhole chain ID and EVM chain ID
    ///      so that governance can resume. Only callable when `isFork()` returns true.
    struct RecoverChainId {
        /// @notice Identifies the Wormhole sub-module this governance action targets.
        /// @dev Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice The governance action type code. Must be 5 for a chain ID recovery.
        uint8 action;
        /// @notice The EVM chain ID (`block.chainid`) of the target forked chain.
        /// @dev Used to ensure this VAA is only executed on the intended forked chain.
        uint256 evmChainId;
        /// @notice The new Wormhole chain ID to assign to the forked chain.
        uint16 newChainId;
    }

    /// @notice Parses a serialized contract upgrade payload (governance action 1).
    /// @dev Parse a contract upgrade (action 1) with minimal validation
    ///      Validates that the action code is 1 and that the payload length is exact.
    /// @param encodedUpgrade The raw binary governance VAA payload bytes.
    /// @return cu The parsed ContractUpgrade struct.
    function parseContractUpgrade(
        bytes memory encodedUpgrade
    ) public pure returns (ContractUpgrade memory cu) {
        uint index = 0;

        cu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        cu.action = encodedUpgrade.toUint8(index);
        index += 1;

        require(cu.action == 1, "invalid ContractUpgrade");

        cu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        cu.newContract = address(
            uint160(uint256(encodedUpgrade.toBytes32(index)))
        );
        index += 32;

        require(encodedUpgrade.length == index, "invalid ContractUpgrade");
    }

    /// @notice Parses a serialized guardian set upgrade payload (governance action 2).
    /// @dev Parse a guardianSet upgrade (action 2) with minimal validation
    ///      Validates the action code is 2 and that the payload length matches the expected size
    ///      for the number of guardian keys.
    /// @param encodedUpgrade The raw binary governance VAA payload bytes.
    /// @return gsu The parsed GuardianSetUpgrade struct.
    function parseGuardianSetUpgrade(
        bytes memory encodedUpgrade
    ) public pure returns (GuardianSetUpgrade memory gsu) {
        uint index = 0;

        gsu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        gsu.action = encodedUpgrade.toUint8(index);
        index += 1;

        require(gsu.action == 2, "invalid GuardianSetUpgrade");

        gsu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        gsu.newGuardianSetIndex = encodedUpgrade.toUint32(index);
        index += 4;

        uint8 guardianLength = encodedUpgrade.toUint8(index);
        index += 1;

        gsu.newGuardianSet = Structs.GuardianSet({
            keys: new address[](guardianLength),
            expirationTime: 0
        });

        for (uint i = 0; i < guardianLength; i++) {
            gsu.newGuardianSet.keys[i] = encodedUpgrade.toAddress(index);
            index += 20;
        }

        require(encodedUpgrade.length == index, "invalid GuardianSetUpgrade");
    }

    /// @notice Parses a serialized set-message-fee payload (governance action 3).
    /// @dev Parse a setMessageFee (action 3) with minimal validation
    ///      Validates the action code is 3 and that the payload is exactly the expected length.
    /// @param encodedSetMessageFee The raw binary governance VAA payload bytes.
    /// @return smf The parsed SetMessageFee struct.
    function parseSetMessageFee(
        bytes memory encodedSetMessageFee
    ) public pure returns (SetMessageFee memory smf) {
        uint index = 0;

        smf.module = encodedSetMessageFee.toBytes32(index);
        index += 32;

        smf.action = encodedSetMessageFee.toUint8(index);
        index += 1;

        require(smf.action == 3, "invalid SetMessageFee");

        smf.chain = encodedSetMessageFee.toUint16(index);
        index += 2;

        smf.messageFee = encodedSetMessageFee.toUint256(index);
        index += 32;

        require(encodedSetMessageFee.length == index, "invalid SetMessageFee");
    }

    /// @notice Parses a serialized transfer-fees payload (governance action 4).
    /// @dev Parse a transferFees (action 4) with minimal validation
    ///      Validates the action code is 4 and that the payload is exactly the expected length.
    /// @param encodedTransferFees The raw binary governance VAA payload bytes.
    /// @return tf The parsed TransferFees struct.
    function parseTransferFees(
        bytes memory encodedTransferFees
    ) public pure returns (TransferFees memory tf) {
        uint index = 0;

        tf.module = encodedTransferFees.toBytes32(index);
        index += 32;

        tf.action = encodedTransferFees.toUint8(index);
        index += 1;

        require(tf.action == 4, "invalid TransferFees");

        tf.chain = encodedTransferFees.toUint16(index);
        index += 2;

        tf.amount = encodedTransferFees.toUint256(index);
        index += 32;

        tf.recipient = encodedTransferFees.toBytes32(index);
        index += 32;

        require(encodedTransferFees.length == index, "invalid TransferFees");
    }

    /// @notice Parses a serialized recover-chain-id payload (governance action 5).
    /// @dev Parse a recoverChainId (action 5) with minimal validation
    ///      Validates the action code is 5 and that the payload is exactly the expected length.
    ///      This action is only valid on forked chains where `isFork()` returns true.
    /// @param encodedRecoverChainId The raw binary governance VAA payload bytes.
    /// @return rci The parsed RecoverChainId struct.
    function parseRecoverChainId(
        bytes memory encodedRecoverChainId
    ) public pure returns (RecoverChainId memory rci) {
        uint index = 0;

        rci.module = encodedRecoverChainId.toBytes32(index);
        index += 32;

        rci.action = encodedRecoverChainId.toUint8(index);
        index += 1;

        require(rci.action == 5, "invalid RecoverChainId");

        rci.evmChainId = encodedRecoverChainId.toUint256(index);
        index += 32;

        rci.newChainId = encodedRecoverChainId.toUint16(index);
        index += 2;

        require(
            encodedRecoverChainId.length == index,
            "invalid RecoverChainId"
        );
    }
}
