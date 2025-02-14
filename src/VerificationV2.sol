// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/RawDispatcher.sol";
import "./ThresholdVerification.sol";
import "./GuardianSetVerification.sol";

contract VerificationV2 is RawDispatcher, ThresholdVerification, GuardianSetVerification {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	error InvalidDispatchVersion(uint8 version);
	error InvalidModule(bytes32 module);
	error InvalidAction(uint8 action);

	uint8 constant OP_GOVERNANCE = 0x00;
	uint8 constant OP_PULL_GUARDIAN_SETS = 0x01;

	uint8 constant OP_VERIFY = 0x20;
	uint8 constant OP_THRESHOLD_GET_CURRENT = 0x21;
	uint8 constant OP_THRESHOLD_GET = 0x22;
	uint8 constant OP_GUARDIAN_SET_GET_CURRENT = 0x23;
	uint8 constant OP_GUARDIAN_SET_GET = 0x24;

	bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);
	uint8 constant RAW_DISPATCH_PROTOCOL_VERSION = 0x01;

	uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

	function decodeAndVerifyVaa(bytes calldata encodedVaa) internal view returns (VaaBody memory) {
		uint8 version = uint8(encodedVaa[0]);
		if (version == 2) {
			return verifyThresholdVAA(encodedVaa);
		} else if (version == 1) {
			return verifyGuardianSetVAA(encodedVaa);
		} else {
			revert VaaLib.InvalidVersion(version);
		}
	}

	function decodeThresholdKeyUpdatePayload(bytes memory payload) internal pure returns (
		bytes32 module,
		uint8 action,
		uint32 newThresholdIndex,
		address newThresholdAddr,
		uint32 expirationDelaySeconds,
		bytes32[] memory shards
	) {
		uint offset = 0;
		(module, offset) = payload.asBytes32MemUnchecked(offset);
		(action, offset) = payload.asUint8MemUnchecked(offset);
		(newThresholdIndex, offset) = payload.asUint32MemUnchecked(offset);
		(newThresholdAddr, offset) = payload.asAddressMemUnchecked(offset);
		(expirationDelaySeconds, offset) = payload.asUint32MemUnchecked(offset);

		uint8 shardsLength;
		(shardsLength, offset) = payload.asUint8MemUnchecked(offset);
		shards = new bytes32[](shardsLength);
		for (uint8 i = 0; i < shardsLength; i++) {
			(shards[i], offset) = payload.asBytes32MemUnchecked(offset);
		}
	}

	function _exec(bytes calldata data) internal override returns (bytes memory) {
		uint offset = 0;
		uint8 version;
		(version, offset) = data.asUint8CdUnchecked(offset);
		if (version != RAW_DISPATCH_PROTOCOL_VERSION) revert InvalidDispatchVersion(version);

		uint length = data.length;
		while (offset < length) {
			uint8 op;
			(op, offset) = data.asUint8CdUnchecked(offset);
			
			if (op == OP_GOVERNANCE) {
				// Read the VAA and verify it
				uint32 dataLength;
				(dataLength, offset) = data.asUint32CdUnchecked(offset);

				bytes calldata encodedVaa = data[offset:offset + dataLength];
				VaaBody memory vaaBody = decodeAndVerifyVaa(encodedVaa);
				
				// Decode the payload
				(
					bytes32 module,
					uint8 action,
					uint32 newThresholdIndex,
					address newThresholdAddr,
					uint32 expirationDelaySeconds,
					bytes32[] memory shards
				) = decodeThresholdKeyUpdatePayload(vaaBody.payload);

				if (module != MODULE_VERIFICATION_V2) revert InvalidModule(module);
				if (action != ACTION_APPEND_THRESHOLD_KEY) revert InvalidAction(action);
				
				// Append the threshold key
				_appendThresholdKey(newThresholdIndex, newThresholdAddr, expirationDelaySeconds, shards);
			} else if (op == OP_PULL_GUARDIAN_SETS) {
				pullGuardianSets();
			}
		}

		return new bytes(0);
	}

	function _get(bytes calldata data) internal view override returns (bytes memory) {
		uint offset = 0;
		uint8 version;
		(version, offset) = data.asUint8CdUnchecked(offset);
		if (version != RAW_DISPATCH_PROTOCOL_VERSION) revert InvalidDispatchVersion(version);

		bytes memory result;
		uint length = data.length;
		while (offset < length) {
			uint8 op;
			(op, offset) = data.asUint8CdUnchecked(offset);
			
			bytes memory result_entry;
			if (op == OP_VERIFY) {
				uint32 dataLength;
				(dataLength, offset) = data.asUint32CdUnchecked(offset);

				bytes calldata encodedVaa = data[offset:offset + dataLength];
				VaaBody memory vaaBody = decodeAndVerifyVaa(encodedVaa);
				result_entry = abi.encode(vaaBody);
			} else if (op == OP_THRESHOLD_GET_CURRENT) {
				(uint32 thresholdIndex, address thresholdAddr) = getCurrentThresholdInfo();
				result_entry = abi.encodePacked(thresholdIndex, thresholdAddr);
			} else if (op == OP_THRESHOLD_GET) {
				uint32 index;
				(index, offset) = data.asUint32CdUnchecked(offset);
				(uint32 expirationTime, address thresholdAddr) = getPastThresholdInfo(index);
				result_entry = abi.encodePacked(expirationTime, thresholdAddr);
			} else if (op == OP_GUARDIAN_SET_GET_CURRENT) {
				(uint32 guardianSetIndex, address[] memory guardianSetAddrs) = getCurrentGuardianSetInfo();
				result_entry = abi.encodePacked(guardianSetIndex, guardianSetAddrs);
			} else if (op == OP_GUARDIAN_SET_GET) {
				uint32 index;
				(index, offset) = data.asUint32CdUnchecked(offset);
				(uint32 expirationTime, address[] memory guardianSetAddrs) = getGuardianSetInfo(index);
				result_entry = abi.encodePacked(expirationTime, guardianSetAddrs);
			}

			result = abi.encodePacked(result, result_entry);
		}

		return result;
	}
}
