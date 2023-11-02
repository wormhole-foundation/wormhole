// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

/// @title Wormhole Interface
/// @notice This interface declares the necessary functions and events for interacting with the Wormhole network
interface IWormhole {
    /// @dev Represents a set of guardians with their associated keys and expiration time
    struct GuardianSet {
        address[] keys;
        uint32 expirationTime;
    }

    /// @dev Represents a signature produced by a guardian
    struct Signature {
        bytes32 r;
        bytes32 s;
        uint8 v;
        uint8 guardianIndex;
    }

    /// @dev Represents a Wormhole message with necessary metadata and signatures
    struct VM {
        uint8 version;
        uint32 timestamp;
        uint32 nonce;
        uint16 emitterChainId;
        bytes32 emitterAddress;
        uint64 sequence;
        uint8 consistencyLevel;
        bytes payload;

        uint32 guardianSetIndex;
        Signature[] signatures;

        bytes32 hash;
    }

    /// @dev A VAA that instructs an implementation on a specific chain to upgrade itself
    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;

        address newContract;
    }

    /// @dev A VAA that upgrades a GuardianSet
    struct GuardianSetUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;

        GuardianSet newGuardianSet;
        uint32 newGuardianSetIndex;
    }

    /// @dev Contains details for setting a new message fee
    struct SetMessageFee {
        bytes32 module;
        uint8 action;
        uint16 chain;

        uint256 messageFee;
    }

    /// @dev Contains details for transferring collected fees
    struct TransferFees {
        bytes32 module;
        uint8 action;
        uint16 chain;

        uint256 amount;
        bytes32 recipient;
    }

    /// @dev Contains details for recovering a chain ID
    struct RecoverChainId {
        bytes32 module;
        uint8 action;

        uint256 evmChainId;
        uint16 newChainId;
    }

    /// @notice Emitted when a message is published to the Wormhole network
    event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);

    /// @notice Emitted when a contract is upgraded on the Wormhole network
    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    /// @notice Emitted when a new guardian set is added to the Wormhole network
    event GuardianSetAdded(uint32 indexed index);

    /// @notice Publish a message to be attested by the Wormhole network
    /// @param nonce A free integer field that can be used however the developer would like
    /// @param payload The content of the emitted message, an arbitrary byte array
    /// @param consistencyLevel The level of finality to reach before the guardians will observe and attest the emitted event
    /// @return sequence number that is unique and increments for every message for a given emitter (and implicitly chain)
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);

    /// @notice Initializes the contract with the EVM chain ID mapping upon deployment or upgrade 
    /// @dev Should only be called once, upon contract deployment or upgrade
    function initialize() external;

    /// @dev Parses and verifies a verifiable message (VM)
    /// @param encodedVM An encoded VM in bytes
    /// @return vm A decoded VM struct
    /// @return valid True if the VM is valid and false if not
    /// @return reason If the VM is not valid, the reason is contained in this string
    function parseAndVerifyVM(bytes calldata encodedVM) external view returns (VM memory vm, bool valid, string memory reason);

    /// @dev Validates an arbitrary verifiable message (VM) against an arbitrary guardian set
    /// @param vm A decoded VM struct
    /// @return valid True if the VM is valid and false if not
    /// @return reason If the VM is not valid, the reason is contained in this string
    function verifyVM(VM memory vm) external view returns (bool valid, string memory reason);

    /// @dev Verifies signatures against an arbitrary guardian set
    /// @param signatures An array of Signature
    /// @param guardianSet A guardian set used to verify the signatures provided
    /// @return valid True if the signatures are valid and false if not
    /// @return reason If the signatures are not valid, the reason is contained in this string
    function verifySignatures(bytes32 hash, Signature[] memory signatures, GuardianSet memory guardianSet) external pure returns (bool valid, string memory reason);

    /// @dev Parses an encoded VM to a decoded VM struct
    /// @param encodedVM An encoded VM in bytes
    /// @return vm A decoded VM struct
    function parseVM(bytes memory encodedVM) external pure returns (VM memory vm);

    /// @dev Calculates the required quorum for a given number of guardians
    /// @param numGuardians The number of guardians to calculate the quorum for
    /// @return numSignaturesRequiredForQuorum The number of signatures required to achieve quorum
    function quorum(uint numGuardians) external pure returns (uint numSignaturesRequiredForQuorum);

    /// @dev Fetches the guardian set for the given index
    /// @param index The index for the guardian set
    /// @return guardianSet Returns a guardian set
    function getGuardianSet(uint32 index) external view returns (GuardianSet memory);

    /// @dev Retrieves the index of the current guardian set
    /// @return index Returns the guardian set's index
    function getCurrentGuardianSetIndex() external view returns (uint32);

    /// @dev Returns the expiration time for the current guardian set
    function getGuardianSetExpiry() external view returns (uint32);

    /// @dev Checks whether a governance action has already been consumed
    /// @return consumed Returns true if consumed
    function governanceActionIsConsumed(bytes32 hash) external view returns (bool);

    /// @dev Determines if the given contract implementation has been initialized
    /// @param impl The address of the contract implementation
    /// @return initialized Returns true if initialized
    function isInitialized(address impl) external view returns (bool);

    /// @dev Returns the chain ID
    function chainId() external view returns (uint16);

    /// @dev Checks if the current chain is a fork
    function isFork() external view returns (bool);

    /// @dev Returns the governance chain ID
    function governanceChainId() external view returns (uint16);

    /// @dev Returns the address of the governance contract
    function governanceContract() external view returns (bytes32);

    /// @dev Gets the current message fee
    function messageFee() external view returns (uint256);

    /// @dev Gets the EVM chain ID
    function evmChainId() external view returns (uint256);

    /// @dev Fetches the next sequence number for a given emitter address
    function nextSequence(address emitter) external view returns (uint64);

    /// @dev Parses an encoded contract upgrade (action 1) to its structured representation 
    /// @param encodedUpgrade the encoded contract upgrade in bytes
    function parseContractUpgrade(bytes memory encodedUpgrade) external pure returns (ContractUpgrade memory cu);

    /// @dev Parses an encoded guardian set upgrade (action 2) to its structured representation 
    /// @param encodedUpgrade the encoded guardian set upgrade in bytes
    function parseGuardianSetUpgrade(bytes memory encodedUpgrade) external pure returns (GuardianSetUpgrade memory gsu);

    /// @dev Parses an encoded message fee update (action 3) to its structured representation
    function parseSetMessageFee(bytes memory encodedSetMessageFee) external pure returns (SetMessageFee memory smf);

    /// @dev Parses an encoded transfer of fees (action 4) to its structured representation
    function parseTransferFees(bytes memory encodedTransferFees) external pure returns (TransferFees memory tf);

    /// @dev Parses an encoded recover chain ID operation (action 5) to its structured representation
    function parseRecoverChainId(bytes memory encodedRecoverChainId) external pure returns (RecoverChainId memory rci);

    /**
     * @dev Upgrades a contract via Governance VAA/VM
     * @param _vm The encoded VAA/VM data
     */
    function submitContractUpgrade(bytes memory _vm) external;

    /**
     * @dev Sets a `messageFee` via Governance VAA/VM
     * @param _vm The encoded VAA/VM data
     */
    function submitSetMessageFee(bytes memory _vm) external;

    /**
     * @dev Deploys a new `guardianSet` via Governance VAA/VM
     * @param _vm The encoded VAA/VM data
     */
    function submitNewGuardianSet(bytes memory _vm) external;

    /**
     * @dev Submits transfer fees to the recipient via Governance VAA/VM
     * @param _vm The encoded VAA/VM data
     */
    function submitTransferFees(bytes memory _vm) external;

    /**
    * @dev Updates the `chainId` and `evmChainId` on a forked chain via Governance VAA/VM
    * @param _vm The encoded VAA/VM data
    */
    function submitRecoverChainId(bytes memory _vm) external;
}
