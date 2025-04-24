// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import "./ThresholdVerificationState.sol";

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

uint constant VAA_V2_HEADER_SIZE = 1 + 4 + 32 + 32 + 1;

contract ThresholdVerification is ThresholdVerificationState {
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

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
      bytes32 r; bytes32 s; uint8 v;

      (version, offset) = encodedVaa.asUint8CdUnchecked(offset);
      (tssIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
      (r, s, v, offset) = _decodeThresholdSignatureCdUnchecked(encodedVaa, offset);

      // Validate and return the VAA body
      require(version == 2, VaaLib.InvalidVersion(version));

      (address thresholdAddr, uint32 expirationTime) = _getThresholdInfo(tssIndex);
      require(expirationTime > block.timestamp, ThresholdKeyExpired());

      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(offset);
      require(ecrecover(vaaHash, v, r, s) == thresholdAddr, ThresholdSignatureVerificationFailed());

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
  ) internal pure returns (bytes32 r, bytes32 s, uint8 v, uint nextOffset) {
    unchecked {
      (r, offset) = encodedVaa.asBytes32CdUnchecked(offset);
      (s, offset) = encodedVaa.asBytes32CdUnchecked(offset);
      (v, offset) = encodedVaa.asUint8CdUnchecked(offset);
      v += VaaLib.SIGNATURE_RECOVERY_MAGIC;
      return (r, s, v, offset);
    }
  }

  function _decodeThresholdKeyUpdatePayload(bytes calldata payload) internal pure returns (
    uint32 newThresholdIndex,
    address newThresholdAddr,
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
      (newThresholdAddr, offset) = payload.asAddressMemUnchecked(offset);
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
    (shardInfo.tlsKey, offset) = data.asBytes32CdUnchecked(offset);
    return (shardInfo, offset);
  }
}
