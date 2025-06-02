// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";

import {EIP712Encoding} from "./EIP712Encoding.sol";
import {ThresholdVerificationState} from "./ThresholdVerificationState.sol";

contract GuardianRegistryVerification is EIP712Encoding {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error RegistrationMessageExpired();
  error GuardianSignatureVerificationFailed();

  function _verifyRegisterGuardian(
    address[] memory guardianAddrs,
    uint32 guardianSet,
    uint32 expirationTime,
    bytes32 id,
    uint8 guardian,
    bytes32 r, bytes32 s, uint8 v
  ) internal view {
    require(expirationTime > block.timestamp, RegistrationMessageExpired());
    bytes32 digest = getRegisterGuardianDigest(guardianSet, expirationTime, id);

    // Verify the signature
    // We're not doing replay protection with the signature itself so we don't care about
    // verifying only canonical (low s) signatures.
    address signatory = ecrecover(digest, v, r, s);
    require(signatory == guardianAddrs[guardian], GuardianSignatureVerificationFailed());
  }
}
