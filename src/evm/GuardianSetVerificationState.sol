// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {ICoreBridge, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";
import {ExtStore} from "./ExtStore.sol";

contract GuardianSetVerificationState is ExtStore {
  using UncheckedIndexing for address[];

  error InvalidGuardianSetIndex();

	// Core bridge instance
  ICoreBridge private immutable _coreBridge;

  // Guardian set expiration time is stored in an array mapped from index to expiration time
  uint32[] private _guardianSetExpirationTime;

  constructor(
    address coreBridge,
    uint256 pullLimit
  ) {
    _coreBridge = ICoreBridge(coreBridge);
    _pullGuardianSets(pullLimit);
  }

	// Get the guardian addresses for a given guardian set index using the ExtStore
  // On an invalid index, the function will panic
  function _getGuardianSetInfo(uint32 index) internal view returns (
    uint32 expirationTime,
    address[] memory guardianAddrs
  ) {
    require(index < _guardianSetExpirationTime.length, InvalidGuardianSetIndex());
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

  function _getCurrentGuardianSetInfo() internal view returns (uint32 index, address[] memory guardianAddrs) {
    unchecked {
      index = uint32(_guardianSetExpirationTime.length - 1);
      (, guardianAddrs) = _getGuardianSetInfo(index);
    }
  }

  function _pullGuardianSets(uint256 limit) internal {
    unchecked {
      // Get the guardian set lengths for the bridge and the local contract
      uint currentGuardianSetLength = _coreBridge.getCurrentGuardianSetIndex() + 1;
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
      uint upper = (limit == 0 || currentGuardianSetLength - oldGuardianSetLength < limit)
        ? currentGuardianSetLength : oldGuardianSetLength + limit;

      // Pull and append the guardian sets
      for (uint i = oldGuardianSetLength; i < upper; ++i) {
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
    GuardianSet memory guardians = _coreBridge.getGuardianSet(index);
    expirationTime = guardians.expirationTime;

    // Convert the guardian set to a byte array
    // Result is stored in `data`
    // NOTE: The `keys` array is temporary and is invalid after this block
    address[] memory keys = guardians.keys;
    assembly ("memory-safe") {
      data := keys
      mstore(data, mul(mload(data), 32))
    }
  }
}
