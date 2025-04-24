// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";

contract GuardianRegistryVerification {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error RegistrationMessageExpired();
  error GuardianSignatureVerificationFailed();

  mapping (uint32 => bytes32[]) private _guardianTLSKeys;

  function _registerTLSKey(
    uint32 guardianSetIndex,
    uint8 guardianIndex,
    bytes32 tlsKey,
    uint guardianSetSize
  ) internal {
    if (_guardianTLSKeys[guardianSetIndex].length == 0) {
      _guardianTLSKeys[guardianSetIndex] = new bytes32[](guardianSetSize);
    }

    _guardianTLSKeys[guardianSetIndex][guardianIndex] = tlsKey;
  }

  function _getTLSKeys(uint32 guardianSetIndex) internal view returns (bytes32[] memory tlsKeys) {
    return _guardianTLSKeys[guardianSetIndex];
  }

  function _verifyRegisterTLSKey(
    address[] memory guardianAddrs,
    uint32 guardianSetIndex,
    uint32 expirationTime,
    bytes32 tlsKey,
    uint8 guardianIndex,
    bytes32 r, bytes32 s, uint8 v
  ) internal view {
    require(expirationTime > block.timestamp, RegistrationMessageExpired());
    bytes32 dataHash = keccak256(abi.encodePacked(guardianSetIndex, expirationTime, tlsKey));

    // Verify the signature
    address signatory = ecrecover(dataHash, v, r, s);
    address guardian = guardianAddrs[guardianIndex];
    require(signatory == guardian, GuardianSignatureVerificationFailed());
  }
}
