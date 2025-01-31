// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/VaaLib.sol";

contract ThresholdVerification {
	using BytesParsing for bytes;
	using VaaLib for bytes;

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
	function verifyThresholdVAA(IWormhole.VM memory vm) public view returns (bool) {
		require(vm.version == 2, "VM version must be 2 for threshold signatures");
		require(vm.signatures.length == 1, "Signature count must be 1");

		// Get the current threshold info
		( uint32 currentThresholdIndex,
		  address currentThresholdAddr
		) = this.getCurrentThresholdInfo();

		// Get the threshold address
    address thresholdAddr;
		uint32 vmGuardianSetIndex = vm.guardianSetIndex;
    if (vmGuardianSetIndex != currentThresholdIndex) {
			// If the guardian set index is not the current threshold index, we need to get the past threshold info
			// and validate that it is not expired
      (uint32 expirationTime, address addr) = this.getPastThresholdInfo(vmGuardianSetIndex);
      require(addr != address(0), "invalid guardian set");
      require(expirationTime >= block.timestamp, "guardian set has expired");
      thresholdAddr = addr;
    } else {
			// If the guardian set index is the current threshold index, we can use the current threshold info
			thresholdAddr = currentThresholdAddr;
		}

		// Verify the threshold signature
    IWormhole.Signature memory sig = vm.signatures[0];
    require(ecrecover(vm.hash, sig.v, sig.r, sig.s) == thresholdAddr, "threshold signature invalid");
    return true;
	}

	function _appendThresholdKey(uint32 newIndex, address newAddr, uint32 oldExpirationTime) internal {
		// Get the current threshold info and verify the new index is sequential
		(uint32 index, address currentAddr) = this.getCurrentThresholdInfo();
		require(newIndex == index + 1, "non-sequential index");

		// Store the current threshold info in past threshold info
		uint256 oldInfo = (uint256(uint160(currentAddr)) << 32) | uint256(oldExpirationTime);
		_pastThresholdInfo.push(oldInfo);

		// Update the current threshold info
		uint256 newInfo = (uint256(uint160(newAddr)) << 32) | uint256(newIndex);
		_currentThresholdInfo = newInfo;
	}
}
