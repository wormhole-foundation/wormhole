// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
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

      // Validate the VAA version
      require(version == 2, VaaLib.InvalidVersion(version));

      // Validate the threshold key expiration time
      (uint256 pubkey, uint32 expirationTime) = _getThresholdInfo(tssIndex);
      require(expirationTime > block.timestamp, ThresholdKeyExpired());

      // Get the message hash
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(offset);

      // Extract the parity and public key
      uint8 parity = (pubkey & 1) != 0 ? 28 : 27;
      uint256 px = pubkey >> 1;

      // Validate the threshold signature
      bytes32 e = keccak256(abi.encodePacked(r, pubkey, vaaHash));
      bytes32 sp = bytes32(Q - mulmod(s, px, Q));
      bytes32 ep = bytes32(Q - mulmod(uint256(e), px, Q));
      require(sp != 0, ThresholdSignatureVerificationFailed());

      // the ecrecover precompile implementation checks that the `r` and `s`
      // inputs are non-zero (in this case, `px` and `ep`), thus we don't need to
      // check if they're zero.
      address R = ecrecover(sp, parity, bytes32(px), ep);
      require(R != address(0), ThresholdSignatureVerificationFailed());
      bytes32 expected = keccak256(abi.encodePacked(R, pubkey, vaaHash));
      require(e == expected, ThresholdSignatureVerificationFailed());

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

  function _decodeThresholdKeyUpdatePayload(bytes calldata payload) internal pure returns (
    uint32 newThresholdIndex,
    uint256 newThresholdAddr,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) {
    unchecked {
      // Decode the payload
      uint offset = 0;
      uint8 action;
      bytes32 module;
      uint shardCount;

      (module, offset) = payload.asBytes32MemUnchecked(offset);
      (action, offset) = payload.asUint8MemUnchecked(offset);
      (newThresholdIndex, offset) = payload.asUint32MemUnchecked(offset);
      (newThresholdAddr, offset) = payload.asUint256MemUnchecked(offset);
      (expirationDelaySeconds, offset) = payload.asUint32MemUnchecked(offset);
      (shardCount, offset) = payload.asUint8MemUnchecked(offset); // TODO: We should probably pass this in and get it from the guardian set to ensure the mapping is correct

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
}
