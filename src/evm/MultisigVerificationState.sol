// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {ICoreBridge, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {eagerOr} from "wormhole-sdk/Utils.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";

import {ExtStore} from "./ExtStore.sol";

contract MultisigVerificationState is ExtStore {
  using UncheckedIndexing for address[];

  error InvalidGuardianSetIndex();

  // Core bridge instance
  ICoreBridge private immutable _coreBridge;

  // Guardian set expiration time is stored in an array mapped from index to expiration time
  uint32[] private _guardianSetExpirationTime;

  constructor(
    ICoreBridge coreBridge,
    uint256 initGuardianSetIndex,
    uint256 pullLimit
  ) {
    _coreBridge = coreBridge;

    require(initGuardianSetIndex <= _coreBridge.getCurrentGuardianSetIndex());
    // All previous guardian sets will have expiration timestamp 0
    assembly ("memory-safe") {
      sstore(_guardianSetExpirationTime.slot, initGuardianSetIndex)
    }

    _pullGuardianSets(pullLimit);
  }

  // Get the current guardian set index and addresses
  // NOTE: If no guardian sets have been pulled, the function will panic
  function _getCurrentGuardianSetInfo() internal view returns (uint32 index, address[] memory guardianAddrs) {
    unchecked {
      index = uint32(_guardianSetExpirationTime.length - 1);
      (, guardianAddrs) = _getGuardianSetInfo(index);
    }
  }

  // Get the guardian addresses for a given guardian set index
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
      mstore(guardianAddrs, shr(5, mload(guardianAddrs)))
    }
  }

  function _pullGuardianSets(uint256 limit) internal returns (bool isComplete, uint32 currentGuardianSetIndex) {
    unchecked {
      // Get the guardian set lengths for the bridge and the local contract
      currentGuardianSetIndex = _coreBridge.getCurrentGuardianSetIndex();
      uint32 currentGuardianSetLength = currentGuardianSetIndex + 1;
      uint oldGuardianSetLength = _guardianSetExpirationTime.length;

      // If we have already pulled all the guardian sets, return
      if (currentGuardianSetLength == oldGuardianSetLength) return (true, currentGuardianSetIndex);

      // Check if we need to update the current guardian set
      if (oldGuardianSetLength > 0) {
        // Pull and write the current guardian set expiration time
        uint updateIndex = oldGuardianSetLength - 1;
        (, uint32 expirationTime) = _pullGuardianSet(uint32(updateIndex));
        _guardianSetExpirationTime[updateIndex] = expirationTime;
      }

      // Calculate the upper bound of the guardian sets to pull
      uint upper = eagerOr(limit == 0, currentGuardianSetLength - oldGuardianSetLength < limit)
        ? currentGuardianSetLength : oldGuardianSetLength + limit;

      // Pull and append the guardian sets
      for (uint i = oldGuardianSetLength; i < upper; i++) {
        // Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
        (bytes memory data, uint32 expirationTime) = _pullGuardianSet(uint32(i));
        _guardianSetExpirationTime.push(expirationTime);
        _extWrite(data);
      }

      return (_guardianSetExpirationTime.length == currentGuardianSetLength, currentGuardianSetIndex);
    }
  }
	
  function _pullGuardianSet(uint32 index) private view returns (
    bytes memory data,
    uint32 expirationTime
  ) {
    // Get the guardian set from the core bridge
    // NOTE: The expiration time is copied from the core bridge,
    //       so any invalid guardian set will already be invalidated
    GuardianSet memory guardians = _coreBridge.getGuardianSet(index);
    expirationTime = guardians.expirationTime;

    // Convert the guardian set to a byte array
    // Result is stored in `data`
    // NOTE: The `keys` array is temporary and is invalid after this block
    address[] memory keys = guardians.keys;
    assembly ("memory-safe") {
      data := keys
      mstore(data, shl(5, mload(data)))
    }
  }
}
