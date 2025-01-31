// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "./RawDispatcher.sol";
import "./ThresholdCore.sol";
import "./GuardianSetCore.sol";

contract VerificationV2 is RawDispatcher, ThresholdVerification, GuardianSetVerification {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	uint8 constant OP_GOVERNANCE = 0x00;
	uint8 constant OP_PULL_GUARDIAN_SETS = 0x01;

	uint8 constant OP_VERIFY = 0x20;
	uint8 constant OP_THRESHOLD_GET_CURRENT = 0x21;
	uint8 constant OP_THRESHOLD_GET = 0x22;
	uint8 constant OP_GUARDIAN_SET_GET_CURRENT = 0x23;
	uint8 constant OP_GUARDIAN_SET_GET = 0x24;

	bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

	uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

	function decodeAndVerifyVaa(bytes calldata encodedVaa) internal view returns (bool verified, IWormhole.VM memory vm) {
		vm = encodedVaa.decodeVmStructCd();

		if (vm.version == 2) {
			verified = verifyThresholdVAA(vm);
		} else if (vm.version == 1) {
			verified = verifyGuardianSetVAA(vm);
		} else {
			revert("Unsupported VAA version");
		}
	}

	function decodeThresholdVaaPayload(bytes memory payload) internal pure returns (
		bytes32 module,
		uint8 action,
		uint8 newThresholdIndex,
		address newThresholdAddr,
		uint32 oldExpirationTime
	) {
		uint offset = 0;
		(module, offset) = payload.asBytes32MemUnchecked(offset);
		(action, offset) = payload.asUint8MemUnchecked(offset);
		(newThresholdIndex, offset) = payload.asUint8MemUnchecked(offset);
		(newThresholdAddr, offset) = payload.asAddressMemUnchecked(offset);
		(oldExpirationTime, offset) = payload.asUint32MemUnchecked(offset);
	}

	function _exec(bytes calldata data) internal override returns (bytes memory) {
		uint offset = 0;
		uint8 version;
		(version, offset) = data.asUint8CdUnchecked(offset);
		require(version == RawDispatcher.VERSION, "invalid version");

		uint length = data.length;
		while (offset < length) {
			uint8 op;
			(op, offset) = data.asUint8CdUnchecked(offset);
			
			if (op == OP_GOVERNANCE) {
				// Read the VAA and verify it
				uint32 dataLength;
				(dataLength, offset) = data.asUint32CdUnchecked(offset);

				bytes calldata encodedVaa = data[offset:offset + dataLength];
				(bool verified, IWormhole.VM memory vm) = decodeAndVerifyVaa(encodedVaa);
				require(verified, "invalid threshold vaa");
				
				// Decode the payload
				(bytes32 module, uint8 action, uint8 newThresholdIndex, address newThresholdAddr, uint32 oldExpirationTime) = decodeThresholdVaaPayload(vm.payload);
				require(module == MODULE_VERIFICATION_V2, "invalid module");
				require(action == ACTION_APPEND_THRESHOLD_KEY, "invalid action");
				
				// Append the threshold key
				_appendThresholdKey(newThresholdIndex, newThresholdAddr, oldExpirationTime);
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
		require(version == RawDispatcher.VERSION, "invalid version");

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
				(bool verified,) = decodeAndVerifyVaa(encodedVaa);
				result_entry = abi.encodePacked(verified);
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
