// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import "./WormholeVerifier.sol";

contract ThresholdVerification is WormholeVerifier {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	error InvalidVaa(bytes encodedVaa);
	error InvalidSignatureCount(uint8 count);

	// Current threshold info is stord in a single slot
	// Format:
	//   index (32 bits)
	//   address (160 bits)
	uint256 private _currentThresholdInfo;

	// Past threshold info is stored in an array
	// Format:
	//   expiration time (32 bits)
	//   address (160 bits)
	uint256[] private _pastThresholdInfo;
	
	bytes32[][] private _shards;

	// Get the current threshold signature info
	function getCurrentThresholdInfo() public view returns (uint32 index, address addr) {
		return (
			uint32(_currentThresholdInfo),
			address(uint160(_currentThresholdInfo >> 32))
		);
	}

	// Get the past threshold signature info
	function getPastThresholdInfo(uint32 index) public view returns (uint32 expirationTime, address addr) {
		uint256 info = _pastThresholdInfo[index];
		return (
			uint32(info),
			address(uint160(info >> 32))
		);
	}

	// Verify a threshold signature VAA
	function verifyThresholdVAA(bytes calldata encodedVaa) public view returns (VaaBody memory) {
		unchecked {
			// Check the VAA version
			uint offset = 0;
			uint8 version;
			(version, offset) = encodedVaa.asUint8CdUnchecked(offset);
			if (version != 2) revert VaaLib.InvalidVersion(version);

			// Decode the guardian set index
			uint32 guardianSetIndex;
			(guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);

			// Get the current threshold info
			( uint32 currentThresholdIndex,
				address currentThresholdAddr
			) = this.getCurrentThresholdInfo();

			// Get the threshold address
			address thresholdAddr;
			if (guardianSetIndex != currentThresholdIndex) {
				// If the guardian set index is not the current threshold index, we need to get the past threshold info
				// and validate that it is not expired
				(uint32 expirationTime, address addr) = this.getPastThresholdInfo(guardianSetIndex);
				require(addr != address(0), "invalid guardian set");
				require(expirationTime >= block.timestamp, "guardian set has expired");
				thresholdAddr = addr;
			} else {
				// If the guardian set index is the current threshold index, we can use the current threshold info
				thresholdAddr = currentThresholdAddr;
			}

			// Calculate the VAA hash
			uint envelopeOffset = offset + VaaLib.GUARDIAN_SIGNATURE_SIZE;
			bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);

			// Decode the guardian signature
			uint guardianIndex; bytes32 r; bytes32 s; uint8 v;
			(guardianIndex, r, s, v, offset) = encodedVaa.decodeGuardianSignatureCdUnchecked(offset);

			// Verify the threshold signature
			if (ecrecover(vaaHash, v, r, s) != thresholdAddr) revert VerificationFailed();
		
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

	function _appendThresholdKey(uint32 newIndex, address newAddr, uint32 expirationDelaySeconds, bytes32[] memory shards) internal {
		// Get the current threshold info and verify the new index is sequential
		(uint32 index, address currentAddr) = this.getCurrentThresholdInfo();
		require(newIndex == index + 1, "non-sequential index");

		// Store the current threshold info in past threshold info
		uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
		uint256 oldInfo = (uint256(uint160(currentAddr)) << 32) | uint256(expirationTime);
		_pastThresholdInfo.push(oldInfo);

		// Update the current threshold info
		uint256 newInfo = (uint256(uint160(newAddr)) << 32) | uint256(newIndex);
		_currentThresholdInfo = newInfo;

		// Store the shards
		_shards.push(shards);
	}
}
