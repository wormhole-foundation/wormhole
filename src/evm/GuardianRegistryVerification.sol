// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./GuardianSetVerification.sol";
import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import "./WormholeVerifier.sol";

error RegistrationMessageExpired();

contract GuardianRegistryVerification {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  mapping (uint32 => bytes32[]) private _guardianTLSKeys;

  function registerTLSKey(
    uint32 guardianSetIndex,
    uint8 guardianIndex,
    bytes32 tlsKey,
    uint guardianSetSize
  ) internal {
    unchecked {
      if (_guardianTLSKeys[guardianSetIndex].length == 0) {
        _guardianTLSKeys[guardianSetIndex] = new bytes32[](guardianSetSize);
      }
      _guardianTLSKeys[guardianSetIndex][guardianIndex] = tlsKey;
    }
  }

  function getTLSKeys(uint32 guardianSetIndex) public view returns (bytes32[] memory tlsKeys) {
    unchecked {
      return _guardianTLSKeys[guardianSetIndex];
    }
  }

  function verifyRegisterTLSKey(
    address[] memory guardianAddrs,
    uint32 guardianSetIndex,
    uint32 expirationTime,
    bytes32 tlsKey,
    uint8 guardianIndex,
    bytes32 r, bytes32 s, uint8 v
  ) public view {
    unchecked {
      if (expirationTime < block.timestamp) revert RegistrationMessageExpired();
      bytes32 dataHash = keccak256(abi.encodePacked(guardianSetIndex, expirationTime, tlsKey));
      if (_failsVerificationSingleGuardian(dataHash, guardianIndex, r, s, v, guardianAddrs))
        revert VerificationFailed();
    }
  }

  function _failsVerificationSingleGuardian(
    bytes32 dataHash,
    uint guardianIndex,
    bytes32 r, bytes32 s, uint8 v,
    address[] memory guardians
  ) private pure returns (bool) {
    address signatory = ecrecover(dataHash, v, r, s);
    address guardian = guardians[guardianIndex];
    return signatory != guardian;
  }
}
