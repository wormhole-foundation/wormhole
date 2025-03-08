// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./ExtStore.sol";
import {ICoreBridge, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";

import {CoreBridgeLib} from "wormhole-sdk/libraries/CoreBridge.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";
import "./WormholeVerifier.sol";

contract GuardianSetVerification is ExtStore, WormholeVerifier {
	using BytesParsing for bytes;
	using VaaLib for bytes;
	using UncheckedIndexing for address[];

	ICoreBridge private _coreV1;

	// Guardian set expiration time is stored in an array mapped from index to expiration time
	uint32[] private _guardianSetExpirationTime;

	constructor(
		address coreV1,
		uint pullLimit
	) {
		_coreV1 = ICoreBridge(coreV1);
		pullGuardianSets(pullLimit);
	}

	// Get the guardian addresses for a given guardian set index using the ExtStore
	// On an invalid index, the function will panic
	function getGuardianSetInfo(uint32 index) public view returns (uint32 expirationTime, address[] memory guardianAddrs) {
		expirationTime = _guardianSetExpirationTime[index];
		
		// Read the guardian set data from the ExtStore
		bytes memory data = _extRead(index);

		// Convert the guardian set data to an array of addresses
		// NOTE: The `data` array is temporary and is invalid after this block
		assembly ("memory-safe") {
			guardianAddrs := data
			mstore(guardianAddrs, div(mload(guardianAddrs), 32))
		}
	}

	function getCurrentGuardianSetInfo() public view returns (uint32 index, address[] memory guardianAddrs) {
		unchecked {
			index = uint32(_guardianSetExpirationTime.length - 1);
			(, guardianAddrs) = this.getGuardianSetInfo(index);
		}
	}

	// Verify a guardian set VAA
	function verifyGuardianSetVAA(bytes calldata encodedVaa) public view returns (
		uint32  timestamp,
    uint32  nonce,
    uint16  emitterChainId,
    bytes32 emitterAddress,
    uint64  sequence,
    uint8   consistencyLevel,
    bytes calldata payload
	) {
		unchecked {
			uint offset = VaaLib.checkVaaVersionCd(encodedVaa);
			uint32 guardianSetIndex;
			(guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);

			// Get the guardian set
			(, address[] memory guardians) = this.getGuardianSetInfo(guardianSetIndex);

			// Get the number of guardians
			// NOTE: Optimization puts var on stack thus avoids mload
			uint guardianCount = guardians.length;

			// Get the number of signatures
			uint signatureCount;
			(signatureCount, offset) = encodedVaa.asUint8CdUnchecked(offset);

			// Validate the number of signatures
			// NOTE: This works for empty guardian sets, because the quorum when there are no guardians is 1
			uint quorumCount = CoreBridgeLib.minSigsForQuorum(guardianCount);
			if (signatureCount < quorumCount) revert VerificationFailed();

			// Calculate envelope offset and VAA hash
			uint envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
			bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);

			// Verify the signatures
			// NOTE: Optimization instead of always checking i == 0
			bool isFirstSignature = true;
			uint prevGuardianIndex;
			
			for (uint i = 0; i < signatureCount; ++i) {
				// Decode the guardian index, r, s, and v
				uint guardianIndex; bytes32 r; bytes32 s; uint8 v;
				(guardianIndex, r, s, v, offset) = encodedVaa.decodeGuardianSignatureCdUnchecked(offset);
				
				// Verify the signature
				if (_failsVerification(
					vaaHash,
					guardianIndex,
					r, s, v,
					guardians,
					guardianCount,
					prevGuardianIndex,
					isFirstSignature
				)) revert VerificationFailed();

				prevGuardianIndex = guardianIndex;
				isFirstSignature = false;
			}

			// Decode the VAA body
			return encodedVaa.decodeVaaBodyCd(envelopeOffset);
		}
	}

	function pullGuardianSets(uint limit) public {
		unchecked {
			// Get the guardian set lengths for the bridge and the local contract
			uint currentGuardianSetLength = _coreV1.getCurrentGuardianSetIndex() + 1;
			uint oldGuardianSetLength = _guardianSetExpirationTime.length;
			
			// If we have already pulled all the guardian sets, return
			if (currentGuardianSetLength == oldGuardianSetLength) return;

			// Check if we need to update the current guardian set
			if (oldGuardianSetLength > 0) {
				// Pull and write the current guardian set expiration time
				uint updateIndex = oldGuardianSetLength - 1;
				(, uint32 expirationTime) = _pullGuardianSet(uint32(updateIndex));
				_guardianSetExpirationTime[updateIndex] = expirationTime;
			}

			// Calculate the upper bound of the guardian sets to pull
			uint upper = (limit == 0 || currentGuardianSetLength - oldGuardianSetLength < limit) ? currentGuardianSetLength : oldGuardianSetLength + limit;

			// Pull and append the guardian sets
			for (uint i = oldGuardianSetLength; i < upper; i++) {
				// Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
				(bytes memory data, uint32 expirationTime) = _pullGuardianSet(uint32(i));
				_guardianSetExpirationTime.push(expirationTime);
				_extWrite(data);
			}
		}
	}

	function _pullGuardianSet(uint32 index) private view returns (
		bytes memory data,
		uint32 expirationTime
	) {
		// Get the guardian set from the core bridge
		GuardianSet memory guardians = _coreV1.getGuardianSet(index);

		// Convert the guardian set to a byte array
		// Result is stored in `data`
		// NOTE: The `keys` array is temporary and is invalid after this block
		address[] memory keys = guardians.keys;
		assembly ("memory-safe") {
			data := keys
			mstore(data, mul(mload(data), 32))
		}

		// Return the expiration time
		return (data, guardians.expirationTime);
	}

	function _failsVerification(
		bytes32 vaaHash,
		uint guardianIndex,
		bytes32 r, bytes32 s, uint8 v,
		address[] memory guardians,
		uint guardianCount,
		uint prevGuardianIndex,
		bool isFirstSignature
	) private pure returns (bool) {
		address signatory = ecrecover(vaaHash, v, r, s);
		address guardian = guardians.readUnchecked(guardianIndex);

		// Check that:
		// * the guardian indicies are in strictly ascending order (only after the first signature)
		//     this is itself an optimization to efficiently prevent having the same guardian signature
		//     included twice
		// * that the guardian index is not out of bounds
		// * that the signatory is the guardian
		//
		// The core bridge also includes a separate check that signatory is not the zero address
		//   but this is already covered by comparing that the signatory matches the guardian which
		//   [can never be the zero address](https://github.com/wormhole-foundation/wormhole/blob/1dbe8459b96e182932d0dd5ae4b6bbce6f48cb09/ethereum/contracts/Setters.sol#L20)
		return eagerOr(
			eagerOr(
				!eagerOr(isFirstSignature, guardianIndex > prevGuardianIndex),
				guardianIndex >= guardianCount
			),
			signatory != guardian
		);
	}
}
