// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IWormhole} from "wormhole-sdk/interfaces/IWormhole.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";
import {ExtStore} from "./ExtStore.sol";

contract GuardianSetVerificationState is ExtStore {
  using UncheckedIndexing for address[];

  error InvalidGuardianSetIndex();

	// Core bridge instance
  IWormhole private immutable _coreBridge;

  // Guardian set expiration time is stored in an array mapped from index to expiration time
  uint32[] private _guardianSetExpirationTime;

  constructor(
    address coreBridge,
    uint32 pullLimit
  ) {
    _coreBridge = IWormhole(coreBridge);
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
	
  function _pullGuardianSet(uint32 index) private view returns (
    bytes memory data,
    uint32 expirationTime
  ) {
    // Get the guardian set from the core bridge
    IWormhole.GuardianSet memory guardians = _coreBridge.getGuardianSet(index);
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
