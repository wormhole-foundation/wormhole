// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import {eagerAnd, eagerOr} from "wormhole-sdk/Utils.sol";

import {ThresholdVerificationState, ShardInfo} from "./ThresholdVerificationState.sol";

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

contract ThresholdVerification is ThresholdVerificationState {
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

  // Curve order for secp256k1
  uint256 constant internal Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant internal HALF_Q = Q >> 1;

  error ThresholdKeyExpired();
  error ThresholdSignatureVerificationFailed();
  error InvalidModule();
  error InvalidAction();

  // Verify a threshold signature V2 VAA
  // NOTE: This function does not validate the VAA version is V2!
  function _verifyThresholdVaaHeader(bytes calldata encodedVaa) internal view returns (uint envelopeOffset) {
    unchecked {
      // Decode the VAA header
      uint offset = 1;
      uint32 tssIndex;
      address r; uint256 s;

      (tssIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
      (r, offset) = encodedVaa.asAddressCdUnchecked(offset);
      (s, offset) = encodedVaa.asUint256CdUnchecked(offset);

      // Load threshold key info and validate expiration time
      ThresholdKeyInfo memory info = _getThresholdInfo(tssIndex);

      // Calculate the challenge value
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(offset);
      (uint256 px, uint8 parity) = _decodePubkey(info.pubkey);
      uint256 e = uint256(keccak256(abi.encodePacked(px, parity, vaaHash, r)));

      // Calculate the recovered address
      address recovered = ecrecover(
        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
        bytes32(Q - mulmod(px, s, Q)),
        parity,
        // NOTE: This is range checked in _decodeThresholdKeyUpdatePayload
        bytes32(px),
        bytes32(mulmod(px, e, Q))
      );

      // Verify that none of the preconditions were violated
      // NOTE: s < Q prevents signature malleability
      // NOTE: Non-zero r prevents confusion with ecrecover failure
      // NOTE: Non-zero check on s not needed, see the first argument of ecrecover
      bool validSignature = eagerAnd(r != address(0), s < Q);
      bool validExpiration = eagerOr(info.expirationTime == 0, info.expirationTime > block.timestamp);
      bool validRecovered = r == recovered;
      
      require(eagerAnd(validSignature, eagerAnd(validExpiration, validRecovered)), ThresholdSignatureVerificationFailed());

      return offset;
    }
  }

  // Verify and decode a threshold signature V2 VAA
  // NOTE: This function does not validate the VAA version is V2!
  function _verifyAndDecodeThresholdVaa(bytes calldata encodedVaa) internal view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes calldata payload
  ) {
    uint payloadOffset = _verifyThresholdVaaHeader(encodedVaa);
    return encodedVaa.decodeVaaBodyCd(payloadOffset);
  }

  // Decode a threshold key update payload, given the number of shards(which is the same as the number of guardians for the current guardian set)
  function _decodeThresholdKeyUpdatePayload(bytes calldata payload, uint256 shardCount) internal pure returns (
    uint32 newTSSIndex,
    uint256 newThresholdPubkey,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) {
    unchecked {
      // Decode the payload
      uint offset = 0;
      bytes32 module;
      uint8 action;

      // Headedr
      (module, offset) = payload.asBytes32MemUnchecked(offset);
      (action, offset) = payload.asUint8MemUnchecked(offset);

      // Payload
      (newTSSIndex, offset) = payload.asUint32MemUnchecked(offset);
      (newThresholdPubkey, offset) = payload.asUint256MemUnchecked(offset);
      (expirationDelaySeconds, offset) = payload.asUint32MemUnchecked(offset);
      
      // Verify the module and action
      require(module == MODULE_VERIFICATION_V2, InvalidModule());
      require(action == ACTION_APPEND_THRESHOLD_KEY, InvalidAction());

      // Validate the threshold key is non-zero and less than HALF_Q
      (uint256 px,) = _decodePubkey(newThresholdPubkey);
      require(eagerAnd(px != 0, px <= HALF_Q), InvalidThresholdKeyAddress());

      // Decode shards
      shards = new ShardInfo[](shardCount);
      for (uint i = 0; i < shardCount; i++) {
        (shards[i].shard, offset) = payload.asBytes32CdUnchecked(offset);
        (shards[i].id, offset) = payload.asBytes32CdUnchecked(offset);
      }

      // Validate the length of the payload
      payload.length.checkLength(offset);
    }
  }

  function _decodePubkey(uint256 pubkey) internal pure returns (uint256 px, uint8 parity) {
    parity = uint8((pubkey & 1) + VaaLib.SIGNATURE_RECOVERY_MAGIC);
    px = pubkey >> 1;
  }
}
