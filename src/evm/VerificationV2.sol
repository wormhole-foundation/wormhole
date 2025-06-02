// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import {RawDispatcher} from "wormhole-sdk/RawDispatcher.sol";
import {CHAIN_ID_SOLANA} from "wormhole-sdk/constants/Chains.sol";
import {ThresholdVerification} from "./ThresholdVerification.sol";
import {GuardianSetVerification} from "./GuardianSetVerification.sol";
import {GuardianRegistryVerification} from "./GuardianRegistryVerification.sol";

// Raw dispatch operation IDs for exec
uint8 constant OP_GOVERNANCE = 0x00;
uint8 constant OP_PULL_GUARDIAN_SETS = 0x01;
uint8 constant OP_REGISTER_GUARDIAN = 0x02;

// Raw dispatch operation IDs for get
uint8 constant OP_VERIFY_AND_DECODE_VAA = 0x20;
uint8 constant OP_VERIFY_VAA = 0x21;
uint8 constant OP_THRESHOLD_GET_CURRENT = 0x22;
uint8 constant OP_THRESHOLD_GET = 0x23;
uint8 constant OP_GUARDIAN_SET_GET_CURRENT = 0x24;
uint8 constant OP_GUARDIAN_SET_GET = 0x25;
uint8 constant OP_GUARDIAN_SHARDS_GET = 0x26;

// Emitter address for the VerificationV2 contract
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

contract VerificationV2 is 
  RawDispatcher, ThresholdVerification, GuardianSetVerification, GuardianRegistryVerification
{
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

  error InvalidValue();
  error InvalidOperation(uint8 op);
  error InvalidGovernanceChainId();
  error InvalidGovernanceAddress();

  error GuardianSetIsNotCurrent();

  // FIXME: The initial TSS index should be the latest guardian set index, not passed in!
  constructor(address coreV1, uint256 initGuardianSetIndex, uint256 pullLimit)
    GuardianSetVerification(coreV1, initGuardianSetIndex, pullLimit)
    GuardianRegistryVerification()
  {}

  function _verifyVaa(bytes calldata data) private view {
    (uint8 version, ) = data.asUint8CdUnchecked(0);
    if (version == 2) {
      _verifyThresholdVaaHeader(data);
    } else if (version == 1) {
      _verifyGuardianSetVaaHeader(data);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function _verifyAndDecodeVaa(bytes calldata data) private view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes calldata payload
  ) {
    (uint8 version, ) = data.asUint8CdUnchecked(0);
    if (version == 2) {
      (timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload) = _verifyAndDecodeThresholdVaa(data);
    } else if (version == 1) {
      (timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload,,) = _verifyAndDecodeGuardianSetVaa(data);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function _exec(bytes calldata data) internal override returns (bytes memory) {
    require(msg.value == 0, InvalidValue());

    uint offset = 0;
    while (offset < data.length) {
      uint8 op;
      (op, offset) = data.asUint8CdUnchecked(offset);
      
      if (op == OP_GOVERNANCE) {
        // Read the VAA
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        // Decode and verify the VAA
        // TODO: Might be better to have a custom function to do the decoding here
        // so we don't drop so many fields
        (
          ,
          ,
          uint16 emitterChainId,
          bytes32 emitterAddress,
          ,
          ,
          bytes calldata payload,
          uint32 guardianSetIndex,
          address[] memory guardians
        ) = _verifyAndDecodeGuardianSetVaa(encodedVaa);

        // Verify the emitter
        if (emitterChainId != CHAIN_ID_SOLANA) revert InvalidGovernanceChainId();
        if (emitterAddress != GOVERNANCE_ADDRESS) revert InvalidGovernanceAddress();
        
        // Decode the payload
        (
          uint32 newTSSIndex,
          uint256 newThresholdAddr,
          uint32 expirationDelaySeconds,
          ShardInfo[] memory shards
        ) = _decodeThresholdKeyUpdatePayload(payload, guardians.length);
        
        // Append the threshold key
        _appendThresholdKey(guardianSetIndex, newTSSIndex, newThresholdAddr, expirationDelaySeconds, shards);
      } else if (op == OP_PULL_GUARDIAN_SETS) {
        uint32 limit;
        (limit, offset) = data.asUint32CdUnchecked(offset);

        _pullGuardianSets(limit);
      } else if (op == OP_REGISTER_GUARDIAN) {
        // Decode the payload
        uint32 guardianSet;
        uint32 expirationTime;
        bytes32 guardianId;
        uint8 guardian; bytes32 r; bytes32 s; uint8 v;

        (guardianSet, offset) = data.asUint32CdUnchecked(offset);
        (expirationTime, offset) = data.asUint32CdUnchecked(offset);
        (guardianId, offset) = data.asBytes32CdUnchecked(offset);
        (guardian, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);
        
        // We only allow registrations for the current guardian set
        (uint32 currentSetIndex, address[] memory guardianAddrs) = _getCurrentGuardianSetInfo();
        require(guardianSet == currentSetIndex, GuardianSetIsNotCurrent());
        
        _verifyRegisterGuardian(
          guardianAddrs,
          guardianSet,
          expirationTime,
          guardianId,
          guardian,
          r, s, v
        );
        
        _registerGuardian(guardianSet, guardian, guardianId);
      } else {
        revert InvalidOperation(op);
      }
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return new bytes(0);
  }

  function _get(bytes calldata data) internal view override returns (bytes memory) {
    uint offset = 0;
    bytes memory result;
    while (offset < data.length) {
      uint8 op;
      (op, offset) = data.asUint8CdUnchecked(offset);
      
      if (op == OP_VERIFY_AND_DECODE_VAA) {
        // Read the VAA
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        // Verify and decode the VAA
        (
          uint32 timestamp,
          uint32 nonce,
          uint16 emitterChainId,
          bytes32 emitterAddress,
          uint64 sequence,
          uint8 consistencyLevel,
          bytes calldata payload
        ) = _verifyAndDecodeVaa(encodedVaa);

        result = abi.encodePacked(
          result,
          timestamp,
          nonce,
          emitterChainId,
          emitterAddress,
          sequence,
          consistencyLevel,
          payload
        );
      } else if (op == OP_VERIFY_VAA) {
        // Read the VAA
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        // Verify the VAA
        _verifyVaa(encodedVaa);
      } else if (op == OP_THRESHOLD_GET_CURRENT) {
        (uint256 thresholdAddr, uint32 thresholdIndex) = _getCurrentThresholdInfo();

        result = abi.encodePacked(result, thresholdAddr, thresholdIndex);
      } else if (op == OP_THRESHOLD_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);
        
        (uint256 thresholdAddr, uint32 expirationTime) = _getThresholdInfo(index);
        
        result = abi.encodePacked(result, thresholdAddr, expirationTime);
      } else if (op == OP_GUARDIAN_SET_GET_CURRENT) {
        (uint32 guardianSet, address[] memory guardianSetAddrs) = _getCurrentGuardianSetInfo();
        uint8 guardianCount = uint8(guardianSetAddrs.length);

        result = abi.encodePacked(result, guardianCount, guardianSetAddrs, guardianSet);
      } else if (op == OP_GUARDIAN_SET_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);
        
        (uint32 expirationTime, address[] memory guardianSetAddrs) = _getGuardianSetInfo(index);
        uint8 guardianCount = uint8(guardianSetAddrs.length);
        
        result = abi.encodePacked(result, guardianCount, guardianSetAddrs, expirationTime);
      } else if (op == OP_GUARDIAN_SHARDS_GET) {
        uint32 guardianSet;
        uint8 guardian;
        (guardianSet, offset) = data.asUint32CdUnchecked(offset);
        (guardian, offset) = data.asUint8CdUnchecked(offset);
        
        (uint shardCount, bytes32[] memory shards) = _getShardsRaw(guardianSet);
        
        result = abi.encodePacked(result, uint8(shardCount), shards);
      } else {
        revert InvalidOperation(op);
      }
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return result;
  }

  function verifyVaa(bytes calldata encodedVaa) internal view {
    _verifyVaa(encodedVaa);
  }

  function verifyAndDecodeVaa(bytes calldata encodedVaa) internal view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes calldata payload
  ) {
    return _verifyAndDecodeVaa(encodedVaa);
  }
}