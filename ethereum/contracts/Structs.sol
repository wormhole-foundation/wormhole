// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface Structs {
	/**
	 * @notice Provider struct contains chain and governance information.
	 * @param chainId The Wormhole chain ID.
	 * @param governanceChainId The chain ID of the governance chain.
	 * @param governanceContract The address of the governance contract.
	 */
	struct Provider {
		uint16 chainId; ///< The Wormhole chain ID.
		uint16 governanceChainId; ///< The chain ID of the governance chain.
		bytes32 governanceContract; ///< The address of the governance contract.
	}

	/**
	 * @notice GuardianSet struct contains the set of guardian keys and expiration time.
	 * @param keys The array of guardian addresses.
	 * @param expirationTime The expiration time of the guardian set (in seconds since epoch).
	 */
	struct GuardianSet {
		address[] keys; ///< The array of guardian addresses.
		uint32 expirationTime; ///< The expiration time of the guardian set.
	}

	/**
	 * @notice Signature struct contains the ECDSA signature components and guardian index.
	 * @param r The r value of the signature.
	 * @param s The s value of the signature.
	 * @param v The v value of the signature.
	 * @param guardianIndex The index of the guardian who signed.
	 */
	struct Signature {
		bytes32 r; ///< The r value of the signature.
		bytes32 s; ///< The s value of the signature.
		uint8 v; ///< The v value of the signature.
		uint8 guardianIndex; ///< The index of the guardian who signed.
	}

	/**
	 * @notice VM struct represents a Validator Message.
	 * @param version The VM version.
	 * @param timestamp The timestamp of the VM.
	 * @param nonce The nonce for replay protection.
	 * @param emitterChainId The chain ID of the emitter.
	 * @param emitterAddress The address of the emitter.
	 * @param sequence The sequence number of the message.
	 * @param consistencyLevel The consistency level for the message.
	 * @param payload The message payload.
	 * @param guardianSetIndex The index of the guardian set used.
	 * @param signatures The array of guardian signatures.
	 * @param hash The hash of the VM body.
	 */
	struct VM {
		uint8 version; ///< The VM version.
		uint32 timestamp; ///< The timestamp of the VM.
		uint32 nonce; ///< The nonce for replay protection.
		uint16 emitterChainId; ///< The chain ID of the emitter.
		bytes32 emitterAddress; ///< The address of the emitter.
		uint64 sequence; ///< The sequence number of the message.
		uint8 consistencyLevel; ///< The consistency level for the message.
		bytes payload; ///< The message payload.

		uint32 guardianSetIndex; ///< The index of the guardian set used.
		Signature[] signatures; ///< The array of guardian signatures.

		bytes32 hash; ///< The hash of the VM body.
	}
}
