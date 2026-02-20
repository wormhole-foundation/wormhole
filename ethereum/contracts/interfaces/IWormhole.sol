// contracts/interfaces/IWormhole.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

/// @title IWormhole
/// @notice The primary interface for integrating with the Wormhole core bridge.
/// @dev Integrators should use this interface to interact with the Wormhole proxy contract.
///      See https://docs.wormhole.com/wormhole for full protocol documentation.
interface IWormhole {
    // ─────────────────────────────────────────────────────────────────────────
    // Structs
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Represents a versioned set of guardian keys used to validate VAAs.
    /// @dev Guardian sets are rotated via governance. Old sets expire 24 hours after
    ///      a new one is installed to allow in-flight VAAs to be delivered.
    struct GuardianSet {
        /// @notice Ethereum addresses derived from each guardian's ECDSA public key.
        /// @dev Quorum (2/3 + 1) of these must sign a VAA for it to be considered valid.
        address[] keys;
        /// @notice Unix timestamp after which this guardian set is no longer valid.
        /// @dev 0 means the set does not expire (used for the current active set).
        uint32 expirationTime;
    }

    /// @notice Represents a single guardian's ECDSA signature over a VAA hash.
    struct Signature {
        /// @notice The `r` component of the ECDSA signature.
        bytes32 r;
        /// @notice The `s` component of the ECDSA signature.
        bytes32 s;
        /// @notice The `v` recovery byte of the ECDSA signature (27 or 28).
        uint8 v;
        /// @notice The index of the signing guardian within the guardian set.
        /// @dev Indices must be provided in strictly ascending order when verifying.
        uint8 guardianIndex;
    }

    /// @notice Represents a parsed Wormhole VAA (Verified Action Approval).
    /// @dev The core primitive of the Wormhole protocol. A VAA is a signed attestation
    ///      produced by the guardian network over an on-chain observation (e.g. a published message).
    ///      See https://docs.wormhole.com/wormhole/explore-wormhole/vaa
    struct VM {
        /// @notice VAA format version. Currently always 1.
        /// @dev Not included in the hash — its integrity is NOT protected by guardian signatures.
        uint8 version;
        /// @notice Unix timestamp of the source chain block containing the observed transaction.
        uint32 timestamp;
        /// @notice An arbitrary nonce chosen by the message emitter. Not enforced by the protocol.
        uint32 nonce;
        /// @notice Wormhole chain ID of the chain that emitted this message (NOT the EVM chain ID).
        uint16 emitterChainId;
        /// @notice Address of the emitting contract or account, left-padded to 32 bytes.
        bytes32 emitterAddress;
        /// @notice Monotonically increasing sequence number per emitter address.
        /// @dev Together with emitterChainId and emitterAddress, uniquely identifies a VAA.
        uint64 sequence;
        /// @notice Finality level the guardian network waited for before signing.
        /// @dev Chain-specific. On Ethereum: 200 = instant, 1 = safe, 0 = finalized.
        uint8 consistencyLevel;
        /// @notice The arbitrary application-level payload of the message.
        bytes payload;
        /// @notice Index of the guardian set whose signatures are included.
        uint32 guardianSetIndex;
        /// @notice Guardian ECDSA signatures over the VAA body hash.
        Signature[] signatures;
        /// @notice Double-keccak256 hash of the VAA body (timestamp through payload).
        /// @dev `keccak256(abi.encodePacked(keccak256(body)))`. Used as the unique identifier
        ///      for the observation and as the payload signed by guardians.
        bytes32 hash;
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Governance payload structs
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Governance payload for upgrading the core bridge implementation (action 1).
    struct ContractUpgrade {
        /// @notice Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice Governance action code. Must be 1.
        uint8 action;
        /// @notice Wormhole chain ID of the target chain.
        uint16 chain;
        /// @notice Address of the new implementation contract.
        address newContract;
    }

    /// @notice Governance payload for rotating the active guardian set (action 2).
    struct GuardianSetUpgrade {
        /// @notice Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice Governance action code. Must be 2.
        uint8 action;
        /// @notice Wormhole chain ID of the target chain (0 = all chains).
        uint16 chain;
        /// @notice The new guardian set to install.
        GuardianSet newGuardianSet;
        /// @notice Must be exactly `getCurrentGuardianSetIndex() + 1`.
        uint32 newGuardianSetIndex;
    }

    /// @notice Governance payload for updating the message publishing fee (action 3).
    struct SetMessageFee {
        /// @notice Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice Governance action code. Must be 3.
        uint8 action;
        /// @notice Wormhole chain ID of the target chain.
        uint16 chain;
        /// @notice The new fee in wei callers must pay to `publishMessage`.
        uint256 messageFee;
    }

    /// @notice Governance payload for transferring accumulated fees to a recipient (action 4).
    struct TransferFees {
        /// @notice Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice Governance action code. Must be 4.
        uint8 action;
        /// @notice Wormhole chain ID of the target chain (0 = all chains).
        uint16 chain;
        /// @notice Amount in wei to transfer.
        uint256 amount;
        /// @notice Recipient address, left-padded to 32 bytes.
        bytes32 recipient;
    }

    /// @notice Governance payload for recovering chain IDs on a hard-forked chain (action 5).
    struct RecoverChainId {
        /// @notice Must equal the "Core" module identifier.
        bytes32 module;
        /// @notice Governance action code. Must be 5.
        uint8 action;
        /// @notice Native EVM chain ID (`block.chainid`) of the forked chain.
        uint256 evmChainId;
        /// @notice New Wormhole chain ID to assign to the forked chain.
        uint16 newChainId;
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Events
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Emitted when a message is published via `publishMessage`.
    /// @dev Guardian nodes observe this event and produce a VAA attesting to the message.
    event LogMessagePublished(
        address indexed sender,
        uint64 sequence,
        uint32 nonce,
        bytes payload,
        uint8 consistencyLevel
    );

    /// @notice Emitted when the core bridge implementation contract is upgraded.
    event ContractUpgraded(
        address indexed oldContract,
        address indexed newContract
    );

    /// @notice Emitted when a new guardian set is installed.
    event GuardianSetAdded(uint32 indexed index);

    // ─────────────────────────────────────────────────────────────────────────
    // Core message publishing
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Publishes a message to the Wormhole network.
    /// @dev Emits a `LogMessagePublished` event. Guardian nodes attest this and produce a VAA.
    ///      Must be called with exactly `messageFee()` wei.
    /// @param nonce Arbitrary nonce, not enforced by the protocol (used by integrators for deduplication/batching).
    /// @param payload The application-level message bytes.
    /// @param consistencyLevel Required finality before guardians sign (chain-specific; on Ethereum: 0=finalized, 200=instant).
    /// @return sequence The sequence number assigned to this message for this emitter.
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);

    /// @notice Initializes the implementation contract. Called once after each upgrade.
    /// @dev Sets the EVM chain ID from a hardcoded Wormhole→EVM chain ID mapping.
    function initialize() external;

    // ─────────────────────────────────────────────────────────────────────────
    // VAA parsing and verification
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Parses and fully validates a raw binary VAA in a single call.
    /// @dev Prefer this over calling `parseVM` + `verifyVM` separately.
    /// @param encodedVM The raw binary VAA bytes.
    /// @return vm The parsed VM struct.
    /// @return valid True if the VAA is valid (signatures, quorum, guardian set).
    /// @return reason Reason string if invalid, empty if valid.
    function parseAndVerifyVM(
        bytes calldata encodedVM
    ) external view returns (VM memory vm, bool valid, string memory reason);

    /// @notice Validates a pre-parsed VAA against the active guardian set.
    /// @dev Checks guardian set validity, expiry, quorum, signatures, and hash integrity.
    /// @param vm A pre-parsed VM struct (e.g. from `parseVM`).
    /// @return valid True if the VAA is valid.
    /// @return reason Reason string if invalid, empty if valid.
    function verifyVM(
        VM memory vm
    ) external view returns (bool valid, string memory reason);

    /// @notice Verifies a set of ECDSA guardian signatures over a hash.
    /// @dev Does NOT check quorum or guardian set validity — use `verifyVM` for full validation.
    ///      Returns true for an empty signatures array.
    /// @param hash The VAA body hash that was signed.
    /// @param signatures Guardian signatures to verify. Indices must be ascending.
    /// @param guardianSet The guardian set to verify against.
    /// @return valid True if all signatures are valid for the given guardian set.
    /// @return reason Reason string if invalid, empty if valid.
    function verifySignatures(
        bytes32 hash,
        Signature[] memory signatures,
        GuardianSet memory guardianSet
    ) external pure returns (bool valid, string memory reason);

    /// @notice Parses a raw binary VAA into a VM struct without validation.
    /// @dev The hash field is computed from the body during parsing and can be trusted.
    ///      Call `verifyVM` or `parseAndVerifyVM` to also validate signatures and quorum.
    /// @param encodedVM The raw binary VAA bytes.
    /// @return vm The parsed (but unvalidated) VM struct.
    function parseVM(
        bytes memory encodedVM
    ) external pure returns (VM memory vm);

    /// @notice Returns the minimum number of signatures required for quorum.
    /// @dev Uses a 2/3 + 1 Byzantine fault tolerant threshold. Maximum 255 guardians.
    /// @param numGuardians Total number of guardians in the set.
    /// @return numSignaturesRequiredForQuorum Minimum signatures needed (floor(2/3 * n) + 1).
    function quorum(
        uint numGuardians
    ) external pure returns (uint numSignaturesRequiredForQuorum);

    // ─────────────────────────────────────────────────────────────────────────
    // Getters
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Returns the guardian set at the given version index.
    /// @param index The version number of the guardian set (starts at 0, increments on each rotation).
    /// @return The GuardianSet containing guardian addresses and expiration time.
    function getGuardianSet(
        uint32 index
    ) external view returns (GuardianSet memory);

    /// @notice Returns the index of the currently active guardian set.
    /// @return The current guardian set version number.
    function getCurrentGuardianSetIndex() external view returns (uint32);

    /// @notice Returns the duration (in seconds) old guardian sets remain valid after rotation.
    /// @dev Currently 86400 (24 hours). After this period, VAAs signed by the old set are rejected.
    /// @return The guardian set expiry window in seconds.
    function getGuardianSetExpiry() external view returns (uint32);

    /// @notice Returns whether a governance VAA has already been executed.
    /// @dev Governance actions are tracked by VAA hash to prevent replay attacks.
    /// @param hash The double-keccak256 hash of the governance VAA body.
    /// @return True if the governance action has been executed.
    function governanceActionIsConsumed(
        bytes32 hash
    ) external view returns (bool);

    /// @notice Returns whether a given implementation address has been initialized.
    /// @param impl The implementation contract address to check.
    /// @return True if the implementation has been initialized via `initialize()`.
    function isInitialized(address impl) external view returns (bool);

    /// @notice Returns the Wormhole chain ID of this chain.
    /// @dev This is NOT the EVM chain ID (`block.chainid`). Wormhole maintains its own chain ID registry.
    ///      Example: Ethereum mainnet = 2, BSC = 4, Polygon = 5.
    ///      See https://docs.wormhole.com/wormhole/reference/environments/evm
    ///      Use `evmChainId()` for the native EVM chain ID.
    /// @return The Wormhole chain ID for this deployment.
    function chainId() external view returns (uint16);

    /// @notice Returns whether this chain is a hard fork of another chain.
    /// @dev Returns true when `block.chainid` differs from the stored `evmChainId`.
    ///      On a fork, governance actions from the original chain would otherwise be replayable.
    ///      Use `submitRecoverChainId` to re-synchronize a forked chain.
    /// @return True if a hard fork has been detected.
    function isFork() external view returns (bool);

    /// @notice Returns the Wormhole chain ID of the governance chain.
    /// @dev Governance VAAs must originate from this chain. Currently Solana (Wormhole chain ID 1).
    /// @return The Wormhole chain ID of the governance chain (currently 1).
    function governanceChainId() external view returns (uint16);

    /// @notice Returns the address of the governance contract on the governance chain, left-padded to 32 bytes.
    /// @dev Only governance VAAs emitted by this contract are accepted.
    /// @return The 32-byte governance contract address.
    function governanceContract() external view returns (bytes32);

    /// @notice Returns the fee in wei required to publish a Wormhole message.
    /// @dev Must be paid exactly when calling `publishMessage`. Updated via `submitSetMessageFee` governance.
    /// @return The current message fee in wei.
    function messageFee() external view returns (uint256);

    /// @notice Returns the native EVM chain ID of this chain.
    /// @dev Unlike `chainId()` (Wormhole chain ID), this is `block.chainid` as per EIP-155.
    ///      Used with `isFork()` to detect hard forks.
    /// @return The EVM chain ID (e.g. 1 for Ethereum mainnet).
    function evmChainId() external view returns (uint256);

    /// @notice Returns the next sequence number that will be assigned to a message from the given emitter.
    /// @dev Sequence numbers are monotonically increasing per emitter, starting at 0.
    ///      Together with emitterChainId and emitterAddress, uniquely identifies a VAA.
    /// @param emitter The emitter address to query.
    /// @return The next sequence number for this emitter.
    function nextSequence(address emitter) external view returns (uint64);

    // ─────────────────────────────────────────────────────────────────────────
    // Governance payload parsers
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Parses a serialized contract upgrade governance payload (action 1).
    /// @param encodedUpgrade The raw governance VAA payload bytes.
    /// @return cu The parsed ContractUpgrade struct.
    function parseContractUpgrade(
        bytes memory encodedUpgrade
    ) external pure returns (ContractUpgrade memory cu);

    /// @notice Parses a serialized guardian set upgrade governance payload (action 2).
    /// @param encodedUpgrade The raw governance VAA payload bytes.
    /// @return gsu The parsed GuardianSetUpgrade struct.
    function parseGuardianSetUpgrade(
        bytes memory encodedUpgrade
    ) external pure returns (GuardianSetUpgrade memory gsu);

    /// @notice Parses a serialized set-message-fee governance payload (action 3).
    /// @param encodedSetMessageFee The raw governance VAA payload bytes.
    /// @return smf The parsed SetMessageFee struct.
    function parseSetMessageFee(
        bytes memory encodedSetMessageFee
    ) external pure returns (SetMessageFee memory smf);

    /// @notice Parses a serialized transfer-fees governance payload (action 4).
    /// @param encodedTransferFees The raw governance VAA payload bytes.
    /// @return tf The parsed TransferFees struct.
    function parseTransferFees(
        bytes memory encodedTransferFees
    ) external pure returns (TransferFees memory tf);

    /// @notice Parses a serialized recover-chain-id governance payload (action 5).
    /// @param encodedRecoverChainId The raw governance VAA payload bytes.
    /// @return rci The parsed RecoverChainId struct.
    function parseRecoverChainId(
        bytes memory encodedRecoverChainId
    ) external pure returns (RecoverChainId memory rci);

    // ─────────────────────────────────────────────────────────────────────────
    // Governance actions
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Upgrades the core bridge implementation contract via a governance VAA.
    /// @dev Reverts on forked chains. Use `submitRecoverChainId` first on forks.
    /// @param _vm The raw binary governance VAA authorizing the upgrade.
    function submitContractUpgrade(bytes memory _vm) external;

    /// @notice Updates the message publishing fee via a governance VAA.
    /// @param _vm The raw binary governance VAA authorizing the fee change.
    function submitSetMessageFee(bytes memory _vm) external;

    /// @notice Rotates the active guardian set via a governance VAA.
    /// @dev The new set index must be `getCurrentGuardianSetIndex() + 1`. The old set expires in 24h.
    /// @param _vm The raw binary governance VAA encoding the new guardian set.
    function submitNewGuardianSet(bytes memory _vm) external;

    /// @notice Transfers accumulated message fees to a recipient via a governance VAA.
    /// @param _vm The raw binary governance VAA authorizing the fee transfer.
    function submitTransferFees(bytes memory _vm) external;

    /// @notice Re-synchronizes chain IDs on a hard-forked chain via a governance VAA.
    /// @dev Only callable when `isFork()` returns true.
    /// @param _vm The raw binary governance VAA encoding the new chain IDs.
    function submitRecoverChainId(bytes memory _vm) external;
}
