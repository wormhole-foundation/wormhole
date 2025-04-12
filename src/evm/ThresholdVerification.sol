// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import "./WormholeVerifier.sol";

contract ThresholdVerification {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error InvalidVaa(bytes encodedVaa);
  error InvalidSignatureCount(uint8 count);
  error InvalidGuardianSet();
  error GuardianSetExpired();

  // Current threshold info is stord in a single slot
  // Format:
  //   index (32 bits)
  //   address (160 bits)
  // TODO: Extract this into its own functions
  uint256 private _currentThresholdInfo;

  // Past threshold info is stored in an array
  // Format:
  //   expiration time (32 bits)
  //   address (160 bits)
  uint256[] private _pastThresholdInfo;
  
  bytes32[][] private _shards;

  // Get the current threshold signature info
  function getCurrentThresholdInfo() public view returns (uint32 index, address addr) {
    return _decodeThresholdInfo(_currentThresholdInfo);
  }

  // Get the past threshold signature info
  function getPastThresholdInfo(uint32 index) public view returns (
    uint32 expirationTime,
    address addr
  ) {
    if (index >= _pastThresholdInfo.length) revert InvalidIndex();
    return _decodeThresholdInfo(_pastThresholdInfo[index]);
  }

  // Verify a threshold signature VAA
  function verifyThresholdVAA(bytes calldata encodedVaa) public view returns (
    uint32  timestamp,
    uint32  nonce,
    uint16  emitterChainId,
    bytes32 emitterAddress,
    uint64  sequence,
    uint8   consistencyLevel,
    bytes calldata payload
  ) {
    unchecked {
      // Check the VAA version
      uint offset = 0;
      uint8 version;
      (version, offset) = encodedVaa.asUint8CdUnchecked(offset);
      if (version != 2) revert VaaLib.InvalidVersion(version);

      // Decode the guardian set index
      uint32 guardianSetIndex;
      (guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);

      // Get the current threshold info
      ( uint32 currentThresholdIndex,
        address currentThresholdAddr
      ) = this.getCurrentThresholdInfo();

      // Get the threshold address
      address thresholdAddr;
      if (guardianSetIndex != currentThresholdIndex) {
        // If the guardian set index is not the current threshold index,
        // we need to get the past threshold info and validate that it is not expired
        (uint32 expirationTime, address addr) = this.getPastThresholdInfo(guardianSetIndex);
        if (expirationTime < block.timestamp) revert GuardianSetExpired();
        thresholdAddr = addr;
      } else {
        // If the guardian set index is the current threshold index,
        // we can use the current threshold info
        thresholdAddr = currentThresholdAddr;
      }

      // Decode the guardian signature
      bytes32 r; bytes32 s; uint8 v;
      (r, s, v, offset) = _decodeThresholdSignatureCdUnchecked(encodedVaa, offset);

      // Verify the threshold signature
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(offset);
      if (ecrecover(vaaHash, v, r, s) != thresholdAddr) revert VerificationFailed();

      // Decode the VAA body and return it
      return encodedVaa.decodeVaaBodyCd(offset);
    }
  }

  function _appendThresholdKey(
    uint32 newIndex,
    address newAddr,
    uint32 expirationDelaySeconds,
    bytes32[] memory shards
  ) internal {
    unchecked {
      // Verify the new address is not the zero address
      if (newAddr == address(0)) revert InvalidGuardianSet();

      // Get the current threshold info and verify the new index is sequential
      (uint32 index, address currentAddr) = this.getCurrentThresholdInfo();
      if (newIndex != index + 1) revert InvalidIndex();

      // Store the current threshold info in past threshold info
      uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
      _pastThresholdInfo.push(_encodeThresholdInfo(expirationTime, currentAddr));

      // Update the current threshold info
      _currentThresholdInfo = _encodeThresholdInfo(newIndex, newAddr);

      // Store the shards
      _shards.push(shards);
    }
  }

  function _decodeThresholdSignatureCdUnchecked(
    bytes calldata encodedVaa,
    uint offset
  ) internal pure returns (bytes32 r, bytes32 s, uint8 v, uint nextOffset) {
    (r, offset) = encodedVaa.asBytes32CdUnchecked(offset);
    (s, offset) = encodedVaa.asBytes32CdUnchecked(offset);
    (v, offset) = encodedVaa.asUint8CdUnchecked(offset);
    v += VaaLib.SIGNATURE_RECOVERY_MAGIC;
    return (r, s, v, offset);
  }

  function _decodeThresholdInfo(uint256 info) internal pure returns (uint32 index, address addr) {
    return (
      uint32(info),
      address(uint160(info >> 32))
    );
  }

  function _encodeThresholdInfo(uint32 index, address addr) internal pure returns (uint256 info) {
    return (uint256(uint160(addr)) << 32) | uint256(index);
  }
}
