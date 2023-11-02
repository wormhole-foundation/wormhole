// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

/// @title Wormhole Structs
/// @notice This interface declares several structs that are used in Wormhole .
interface Structs {
	/// @notice The Provider struct.
    /// @param chainId The chain id.
    /// @param governanceChainId The chain ID where the governance contract is deployed.
    /// @param governanceContract The address in bytes of the governance contract.
	struct Provider {
		uint16 chainId;
		uint16 governanceChainId;
		bytes32 governanceContract;
	}

	/// @notice The Guardian Set is a set of guardians that are responsible for validating a message emitted from the core contracts.
	/// @param keys An array of addresses representing the public keys of the guardians.
    /// @param expirationTime The timestamp at which this guardian set expires and can be replaced by a new set.
	struct GuardianSet {
		address[] keys;
		uint32 expirationTime;
	}

	/// @notice Represents a signature produced by a guardian.
    /// @param r The 'r' portion of the ECDSA signature.
    /// @param s The 's' portion of the ECDSA signature.
    /// @param v The recovery byte of the ECDSA signature.
    /// @param guardianIndex The index of the guardian in the current guardian set that produced this signature.
	struct Signature {
		bytes32 r;
		bytes32 s;
		uint8 v;
		uint8 guardianIndex;
	}

	/// @notice Describes a verifiable message (VM) in the Wormhole protocol.
    /// @param version The version of the VM structure.
    /// @param timestamp The timestamp when the VM was produced.
    /// @param nonce A nonce to ensure uniqueness of the VM.
    /// @param emitterChainId The chain ID from where the message originated.
    /// @param emitterAddress The address in bytes from where the message originated.
    /// @param sequence A number that increments with each message from a given emitter, ensuring ordered processing.
    /// @param consistencyLevel The level of consistency required for the message before it is considered confirmed.
    /// @param payload The actual data content of the message.
    /// @param guardianSetIndex The index of the guardian set responsible for signing the VM.
    /// @param signatures An array of signatures by the guardians.
    /// @param hash The hash of the message contents.
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
}
