// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./ExtStore.sol";
import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/UncheckedIndexing.sol";
import "./WormholeVerifier.sol";

contract GuardianSetVerification is ExtStore, WormholeVerifier {
	using UncheckedIndexing for address[];
	using BytesParsing for bytes;
	using VaaLib for bytes;

	IWormhole private _coreV1;

	// Guardian set expiration time is stored in an array mapped from index to expiration time
	uint32[] private _guardianSetExpirationTime;

	// Get the guardian addresses for a given guardian set index using the ExtStore
	function getGuardianSetInfo(uint32 index) public view returns (uint32 expirationTime, address[] memory guardianAddrs) {
		expirationTime = _guardianSetExpirationTime[index];
		
		bytes memory data = _extRead(index);
		assembly ("memory-safe") {
			guardianAddrs := data
			mstore(guardianAddrs, div(mload(guardianAddrs), 32))
		}
	}

	function getCurrentGuardianSetInfo() public view returns (uint32 index, address[] memory guardianAddrs) {
		// NOTE: Expiration time array is one entry shorter than the guardian set index
		// because the current guardian set is not included in the array(no expiration time set)
		index = uint32(_guardianSetExpirationTime.length - 1);
		uint32 expirationTime;
		(expirationTime, guardianAddrs) = this.getGuardianSetInfo(index);
		return (index, guardianAddrs);
	}

	// Verify a guardian set VAA
	function verifyGuardianSetVAA(bytes calldata encodedVaa) public view returns (VaaBody memory) {
		unchecked {
			uint offset = VaaLib.checkVaaVersionCd(encodedVaa);
			uint32 guardianSetIndex;
			(guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);

			// Get the guardian set
			(, address[] memory guardians) = this.getGuardianSetInfo(guardianSetIndex);

			uint guardianCount = guardians.length; //optimization puts var on stack thus avoids mload

			uint signatureCount;
			(signatureCount, offset) = encodedVaa.asUint8CdUnchecked(offset);

			// This works for empty guardian sets, because the quorum when there are no guardians is 1
			uint quorumCount = guardianCount * 2 / 3 + 1;
			if (signatureCount < quorumCount) revert VerificationFailed();

			uint envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
			bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);

			bool isFirstSignature = true; //optimization instead of always checking i == 0
			uint prevGuardianIndex;
			for (uint i = 0; i < signatureCount; ++i) {
				uint guardianIndex; bytes32 r; bytes32 s; uint8 v;
				(guardianIndex, r, s, v, offset) = encodedVaa.decodeGuardianSignatureCdUnchecked(offset);
				
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
			VaaBody memory vaaBody;
			(
				vaaBody.envelope.timestamp,
				vaaBody.envelope.nonce,
				vaaBody.envelope.emitterChainId,
				vaaBody.envelope.emitterAddress,
				vaaBody.envelope.sequence,
				vaaBody.envelope.consistencyLevel,
				vaaBody.payload
			) = encodedVaa.decodeVaaBodyCd(envelopeOffset);

			return vaaBody;
		}
	}

	function pullGuardianSets() public {
		// For each guardian set after the current one
		uint32 coreGuardianSetIndex = _coreV1.getCurrentGuardianSetIndex();
		for (uint32 i = uint32(_guardianSetExpirationTime.length); i <= coreGuardianSetIndex; ++i) {
			// Pull the guardian set from the core V1 contract
			IWormhole.GuardianSet memory guardians = _coreV1.getGuardianSet(i);

			// Convert the guardian set to a byte array
			address[] memory keys = guardians.keys;
			bytes memory data;
			assembly ("memory-safe") {
				data := keys
				mstore(data, mul(mload(data), 32))
			}

			// Write the guardian set to the ExtStore and verify the index
			uint32 extIndex = uint32(_extWrite(data));
			require(extIndex == i, "ext index mismatch");

			// Set the expiration time for the guardian set
			_guardianSetExpirationTime.push(guardians.expirationTime);
		}
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
    //check that:
    // * the guardian indicies are in strictly ascending order (only after the first signature)
    //     this is itself an optimization to efficiently prevent having the same guardian signature
    //     included twice
    // * that the guardian index is not out of bounds
    // * that the signatory is the guardian
    //
    // the core bridge also includes a separate check that signatory is not the zero address
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

	function eagerOr(bool lhs, bool rhs) private pure returns (bool ret) {
		assembly ("memory-safe") {
			ret := or(lhs, rhs)
		}
	}
}
