// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./ExtStore.sol";
import "wormhole-sdk/interfaces/IWormhole.sol";

contract GuardianSetVerification is ExtStore {
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
	function verifyGuardianSetVAA(IWormhole.VM memory vm) public view returns (bool) {
		unchecked {
			// Get the guardian set info
			uint32 vmGuardianSetIndex = vm.guardianSetIndex;
			(uint32 currentGuardianSetIndex, address[] memory guardianAddrs) = this.getCurrentGuardianSetInfo();
			uint sigCount = vm.signatures.length;
			uint guardianCount = guardianAddrs.length;

			// Validate the guardian set
			require(guardianCount != 0, "invalid guardian set");
			require(sigCount > guardianCount * 2 / 3, "no quorum");
			if (vmGuardianSetIndex != currentGuardianSetIndex) {
				require(
					_guardianSetExpirationTime[vmGuardianSetIndex] >= block.timestamp,
					"guardian set has expired"
				);
			}

			// Verify the signatures
			int lastIndex = -1;
			for (uint i = 0; i < sigCount; ++i) {
				IWormhole.Signature memory sig = vm.signatures[i];
				uint idx = sig.guardianIndex;
				require(int(idx) > lastIndex, "signature indices must be ascending");
				require(ecrecover(vm.hash, sig.v, sig.r, sig.s) == guardianAddrs[idx], "VM signature invalid");
				lastIndex = int(idx);
			}

			require(lastIndex < int(guardianCount), "guardian index out of bounds"); // FIXME: Do we need this check? if idx > guardianAddrs.length, it will revert already
			return true;
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
}
