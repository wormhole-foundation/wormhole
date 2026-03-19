// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

/// @title Structs
/// @notice Defines the core data structures used throughout the Wormhole core bridge contracts.
interface Structs {
    /// @notice Identifies the Wormhole provider (chain) configuration.
    /// @dev This is stored in contract state and set during initialization.
    struct Provider {
        /// @notice The Wormhole chain ID of this chain.
        /// @dev This is NOT the same as the EVM chain ID (block.chainid). Wormhole maintains its own
        ///      chain ID registry. See https://wormhole.com/docs/products/reference/chain-ids/
        ///      for the full list of Wormhole chain IDs.
        uint16 chainId;
        /// @notice The Wormhole chain ID of the chain that hosts the governance contract.
        /// @dev Currently Solana (Wormhole chain ID = 1).
        uint16 governanceChainId;
        /// @notice The address of the governance contract on the governance chain, left-padded to 32 bytes.
        /// @dev On Solana this is the Wormhole program's governance account (0x4 left-padded).
        bytes32 governanceContract;
    }

    /// @notice Represents a versioned set of guardian keys used to validate VAAs.
    /// @dev Guardian sets are rotated via governance actions (`submitNewGuardianSet`). Old sets
    ///      expire 24 hours after a new set is installed to allow in-flight VAAs to be delivered.
    struct GuardianSet {
        /// @notice The Ethereum addresses derived from each guardian's ECDSA public key.
        /// @dev Quorum (2/3 + 1) of these addresses must sign a VAA for it to be considered valid.
        address[] keys;
        /// @notice Unix timestamp after which this guardian set is no longer valid.
        /// @dev Set to `block.timestamp + 86400` (24 hours) when a new guardian set is installed.
        ///      A value of 0 means the set does not expire (used for the current active set).
        uint32 expirationTime;
    }

    /// @notice Represents a single guardian's ECDSA signature over a VAA hash.
    struct Signature {
        /// @notice The `r` component of the ECDSA signature.
        bytes32 r;
        /// @notice The `s` component of the ECDSA signature.
        bytes32 s;
        /// @notice The `v` recovery byte of the ECDSA signature (normalized to 27 or 28).
        uint8 v;
        /// @notice The index of the signing guardian within the guardian set.
        /// @dev Used to look up the expected signer address in `GuardianSet.keys`.
        ///      Indices must be provided in strictly ascending order.
        uint8 guardianIndex;
    }

    /// @notice Represents a parsed Wormhole VAA (Verified Action Approval).
    /// @dev A VAA is the core primitive of the Wormhole protocol — a signed attestation
    ///      produced by the guardian network. See https://docs.wormhole.com/wormhole/explore-wormhole/vaa
    struct VM {
        /// @notice The VAA format version. Currently always 1.
        /// @dev Note: the version field is NOT included in the hash and its integrity is not
        ///      protected by guardian signatures. Do not rely on this field for security decisions.
        uint8 version;
        /// @notice The Unix timestamp of the block in which the source transaction was included.
        uint32 timestamp;
        /// @notice An arbitrary nonce chosen by the message emitter.
        /// @dev Used by integrators for message deduplication or batching. Not enforced by the protocol.
        uint32 nonce;
        /// @notice The Wormhole chain ID of the chain that emitted this message.
        /// @dev This is the Wormhole chain ID, NOT the EVM chain ID.
        uint16 emitterChainId;
        /// @notice The address of the contract or account that emitted this message, left-padded to 32 bytes.
        bytes32 emitterAddress;
        /// @notice A monotonically increasing sequence number per emitter address.
        /// @dev Together with `emitterChainId` and `emitterAddress`, this uniquely identifies a VAA.
        ///      Integrators use this for replay protection on the destination chain.
        uint64 sequence;
        /// @notice The finality level required before guardians will sign this message.
        /// @dev The interpretation is chain-specific. On Ethereum: 0 = finalized, 1 = safe, 200 = instant.
        ///      See https://docs.wormhole.com/wormhole/reference/glossary#consistency-level
        uint8 consistencyLevel;
        /// @notice The arbitrary application-level payload of the message.
        bytes payload;
        /// @notice The index of the guardian set whose signatures are included in this VAA.
        uint32 guardianSetIndex;
        /// @notice The list of guardian signatures over the VAA body hash.
        Signature[] signatures;
        /// @notice The double-keccak256 hash of the VAA body (timestamp through payload).
        /// @dev Computed as `keccak256(abi.encodePacked(keccak256(body)))`. Used as the unique
        ///      identifier of the observation and as the signing payload for guardians.
        bytes32 hash;
    }
}
