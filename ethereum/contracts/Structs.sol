// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface Structs {
	struct Provider {
		uint16 chainId;
		uint16 governanceChainId;
		bytes32 governanceContract;
	}

	struct GuardianSet {
		address[] keys;
		uint32 expirationTime;
	}

	struct Signature {
		bytes32 r;
		bytes32 s;
		uint8 v;
		uint8 guardianIndex;
	}

	struct Header {
		uint32 guardianSetIndex;
		Signature[] signatures;
		bytes32 hash;
	}

	struct VM {
		uint8 version; // version = 1 or 3
		// The following fields constitute an Observation. For compatibility
		// reasons we keep the representation inlined.
		uint32 timestamp;
		uint32 nonce;
		uint16 emitterChainId;
		bytes32 emitterAddress;
		uint64 sequence;
		uint8 consistencyLevel;
		bytes payload;
		// End of observation

		// The following fields constitute a Header. For compatibility
		// reasons we keep the representation inlined.
		uint32 guardianSetIndex;
		Signature[] signatures;
		// Computed value
		bytes32 hash;
	}

	struct VM2 {
		uint8 version; // version = 2

		// Inlined Header
		uint32 guardianSetIndex;
		Signature[] signatures;

		// Array of Observation hashes
		bytes32[] hashes;

		// Computed Batch Hash - `hash(version, hash(hash(Observation1), hash(Observation2), ...))`
		bytes32 hash;

		// Array of observation bytes with prepended version 3
		bytes[] observations;
	}
}