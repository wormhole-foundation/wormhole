// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import {GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import "./ThresholdVerificationState.sol";

contract ThresholdVerification is ThresholdVerificationState {
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

  // Module ID for the VerificationV2 contract, ASCII "TSS"
  bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

  // Action ID for appending a threshold key
  uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

  // Curve order for secp256k1
  uint256 constant public Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant public HALF_Q = Q >> 1;

  error ThresholdKeyExpired();
  error ThresholdSignatureVerificationFailed();
  error InvalidModule(bytes32 module);
  error InvalidAction(uint8 action);

  // Verify a threshold signature VAA
  function _verifyThresholdVaaHeader(bytes calldata encodedVaa) internal view returns (uint envelopeOffset) {
    unchecked {
      // Decode the VAA header
      uint offset = 0;
      uint8 version;
      uint32 tssIndex;
      address r; uint256 s;

      (version, offset) = encodedVaa.asUint8CdUnchecked(offset);
      (tssIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
      (r, s, offset) = _decodeThresholdSignatureCdUnchecked(encodedVaa, offset);

      // Validate the VAA version and threshold signature is in range
      // NOTE: s < Q prevents signature malleability
      // NOTE: Non-zero r prevents confusion with ecrecover failure
      // NOTE: Non-zero check on s not needed, see the first argument of ecrecover
      require(version == 2, VaaLib.InvalidVersion(version));
      require(s < Q && r != address(0), ThresholdSignatureVerificationFailed());

      // Load threshold key info and validate expiration time
      (uint256 pubkey, uint32 expirationTime) = _getThresholdInfo(tssIndex);
      require(expirationTime > block.timestamp, ThresholdKeyExpired());

      // Calculate the challenge value
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(offset);
      (uint256 px, bool parity) = _decodePubkey(pubkey);
      uint256 e = uint256(keccak256(abi.encodePacked(px, parity, vaaHash, r)));

      // Verify the recovered address matches the threshold signature r
      address recovered = ecrecover(
        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
        bytes32(Q - mulmod(px, s, Q)),
        parity ? 28 : 27,
        // NOTE: This is checked non-zero in _decodeThresholdKeyUpdatePayload
        bytes32(px),
        bytes32(mulmod(px, e, Q))
      );
      require(r == recovered, ThresholdSignatureVerificationFailed());

      return offset;
    }
  }

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

  function _decodeThresholdSignatureCdUnchecked(
    bytes calldata encodedVaa,
    uint offset
  ) internal pure returns (address r, uint256 s, uint nextOffset) {
    unchecked {
      (r, offset) = encodedVaa.asAddressCdUnchecked(offset);
      (s, offset) = encodedVaa.asUint256CdUnchecked(offset);
      return (r, s, offset);
    }
  }

  function _decodeThresholdKeyUpdatePayload(bytes calldata payload, uint256 shardCount) internal pure returns (
    uint32 newTSSIndex,
    uint256 newThresholdPubkey,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) {
    unchecked {
      // Decode the payload
      uint offset = 0;
      uint8 action;
      bytes32 module;

      // Headedr
      (module, offset) = payload.asBytes32MemUnchecked(offset);
      (action, offset) = payload.asUint8MemUnchecked(offset);

      // Payload
      (newTSSIndex, offset) = payload.asUint32MemUnchecked(offset);
      (newThresholdPubkey, offset) = payload.asUint256MemUnchecked(offset);
      (expirationDelaySeconds, offset) = payload.asUint32MemUnchecked(offset);

      // Validate the threshold key is non-zero and less than HALF_Q
      (uint256 px,) = _decodePubkey(newThresholdPubkey);
      require(px != 0, InvalidThresholdKeyAddress());
      require(px <= HALF_Q, InvalidThresholdKeyAddress());

      // Decode shards
      shards = new ShardInfo[](shardCount);
      for (uint i = 0; i < shardCount; i++) {
        (shards[i], offset) = _decodeShardInfo(payload, offset);
      }

      // Validate the length of the payload
      payload.length.checkLength(offset);

      // Verify the module and action
      require(module == MODULE_VERIFICATION_V2, InvalidModule(module));
      require(action == ACTION_APPEND_THRESHOLD_KEY, InvalidAction(action));
    }
  }

  function _decodeShardInfo(bytes calldata data, uint256 offset) internal pure returns (ShardInfo memory shardInfo, uint256 nextOffset) {
    (shardInfo.shard, offset) = data.asBytes32CdUnchecked(offset);
    (shardInfo.id, offset) = data.asBytes32CdUnchecked(offset);
    return (shardInfo, offset);
  }

  function _decodePubkey(uint256 pubkey) internal pure returns (uint256 px, bool parity) {
    parity = (pubkey & 1) != 0;
    px = pubkey >> 1;
  }
}
