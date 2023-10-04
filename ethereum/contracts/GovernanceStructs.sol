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

    enum GovernanceAction {
        UpgradeContract,
        UpgradeGuardianset
    }

    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;

        address newContract;
    }

    struct GuardianSetUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;

        Structs.GuardianSet newGuardianSet;
        uint32 newGuardianSetIndex;
    }

    struct SetMessageFee {
        bytes32 module;
        uint8 action;
        uint16 chain;

        uint256 messageFee;
    }

    struct TransferFees {
        bytes32 module;
        uint8 action;
        uint16 chain;

        uint256 amount;
        bytes32 recipient;
    }

    struct RecoverChainId {
        bytes32 module;
        uint8 action;

        uint256 evmChainId;
        uint16 newChainId;
    }

    /// @dev Parse a contract upgrade (action 1) with minimal validation
    function parseContractUpgrade(bytes memory encodedUpgrade) public pure returns (ContractUpgrade memory cu) {
        uint index = 0;

        cu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        cu.action = encodedUpgrade.toUint8(index);
        index += 1;

        require(cu.action == 1, "invalid ContractUpgrade");

        cu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        cu.newContract = address(uint160(uint256(encodedUpgrade.toBytes32(index))));
        index += 32;

        require(encodedUpgrade.length == index, "invalid ContractUpgrade");
    }

    /// @dev Parse a guardianSet upgrade (action 2) with minimal validation
    function parseGuardianSetUpgrade(bytes memory encodedUpgrade) public pure returns (GuardianSetUpgrade memory gsu) {
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
            keys : new address[](guardianLength),
            expirationTime : 0
        });

        for(uint i = 0; i < guardianLength; i++) {
            gsu.newGuardianSet.keys[i] = encodedUpgrade.toAddress(index);
            index += 20;
        }

        require(encodedUpgrade.length == index, "invalid GuardianSetUpgrade");
    }

    /// @dev Parse a setMessageFee (action 3) with minimal validation
    function parseSetMessageFee(bytes memory encodedSetMessageFee) public pure returns (SetMessageFee memory smf) {
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

    /// @dev Parse a transferFees (action 4) with minimal validation
    function parseTransferFees(bytes memory encodedTransferFees) public pure returns (TransferFees memory tf) {
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

    /// @dev Parse a recoverChainId (action 5) with minimal validation
    function parseRecoverChainId(bytes memory encodedRecoverChainId) public pure returns (RecoverChainId memory rci) {
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

        require(encodedRecoverChainId.length == index, "invalid RecoverChainId");
    }
}