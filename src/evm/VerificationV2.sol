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
uint8 constant OP_REGISTER_TLS_KEY = 0x02;

// Raw dispatch operation IDs for get
uint8 constant OP_VERIFY = 0x20;
uint8 constant OP_THRESHOLD_GET_CURRENT = 0x21;
uint8 constant OP_THRESHOLD_GET = 0x22;
uint8 constant OP_GUARDIAN_SET_GET_CURRENT = 0x23;
uint8 constant OP_GUARDIAN_SET_GET = 0x24;
uint8 constant OP_GUARDIAN_TLS_GET = 0x25;

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

  constructor(address coreV1, uint pullLimit) 
    ThresholdVerification()
    GuardianSetVerification(coreV1, pullLimit)
    GuardianRegistryVerification() 
  {}

  function _verifyVaa(bytes calldata data, uint offset) private view {
    (uint8 version, ) = data.asUint8CdUnchecked(offset);
    if (version == 2) {
      _verifyThresholdVaa(data, offset);
    } else if (version == 1) {
      _verifyGuardianSetVaa(data, offset);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function _verifyAndDecodeVaa(bytes calldata data, uint offset) private view returns (
    uint32  timestamp,
    uint32  nonce,
    uint16  emitterChainId,
    bytes32 emitterAddress,
    uint64  sequence,
    uint8   consistencyLevel,
    bytes calldata payload,
    uint resultOffset
  ) {
    (uint8 version, ) = data.asUint8CdUnchecked(offset);
    if (version == 2) {
      return _verifyAndDecodeThresholdVaa(data, offset);
    } else if (version == 1) {
      return _verifyAndDecodeGuardianSetVaa(data, offset);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function _exec(bytes calldata data) internal override returns (bytes memory) {
    if (msg.value != 0) revert InvalidValue();

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
          bytes calldata payload
        ) = _verifyAndDecodeGuardianSetVaa(encodedVaa);

        // Verify the emitter
        if (emitterChainId != CHAIN_ID_SOLANA) revert InvalidGovernanceChainId();
        if (emitterAddress != GOVERNANCE_ADDRESS) revert InvalidGovernanceAddress();
        
        // Decode the payload
        (
          uint32 newThresholdIndex,
          address newThresholdAddr,
          uint32 expirationDelaySeconds,
          bytes32[] calldata shards
        ) = _decodeThresholdKeyUpdatePayload(payload);
        
        // Append the threshold key
        _appendThresholdKey(newThresholdIndex, newThresholdAddr, expirationDelaySeconds, shards);
      } else if (op == OP_PULL_GUARDIAN_SETS) {
        uint32 limit;
        (limit, offset) = data.asUint32CdUnchecked(offset);

        _pullGuardianSets(limit);
      } else if (op == OP_REGISTER_TLS_KEY) {
        // Decode the payload
        uint32 guardianSetIndex;
        uint32 expirationTime;
        bytes32 tlsKey;
        uint8 guardianIndex; bytes32 r; bytes32 s; uint8 v;

        // TODO: EIP-712 signature
        (guardianSetIndex, offset) = data.asUint32CdUnchecked(offset);
        (expirationTime, offset) = data.asUint32CdUnchecked(offset);
        (tlsKey, offset) = data.asBytes32CdUnchecked(offset);
        (guardianIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);
        
        // We only allow registrations for the current guardian set
        (uint32 currentSetIndex, address[] memory guardianAddrs) = this.getCurrentGuardianSetInfo();
        require(guardianSetIndex == currentSetIndex, GuardianSetIsNotCurrent());
        
        verifyRegisterTLSKey(
          guardianAddrs,
          guardianSetIndex,
          expirationTime,
          tlsKey,
          guardianIndex,
          r, s, v
        );
        
        registerTLSKey(guardianSetIndex, guardianIndex, tlsKey, guardianAddrs.length);
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
        (uint32 thresholdIndex, address thresholdAddr) = getCurrentThresholdInfo();

        result = abi.encodePacked(result, thresholdIndex, thresholdAddr);
      } else if (op == OP_THRESHOLD_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);
        
        (uint32 expirationTime, address thresholdAddr) = getPastThresholdInfo(index);
        
        result = abi.encodePacked(result, expirationTime, thresholdAddr);
      } else if (op == OP_GUARDIAN_SET_GET_CURRENT) {
        (uint32 guardianSetIndex, address[] memory guardianSetAddrs) = getCurrentGuardianSetInfo();
        
        result = abi.encodePacked(result, guardianSetIndex, guardianSetAddrs);
      } else if (op == OP_GUARDIAN_SET_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);
        
        (uint32 expirationTime, address[] memory guardianSetAddrs) = getGuardianSetInfo(index);
        
        result = abi.encodePacked(result, expirationTime, guardianSetAddrs);
      } else if (op == OP_GUARDIAN_TLS_GET) {
        uint32 guardianSetIndex;
        (guardianSetIndex, offset) = data.asUint32CdUnchecked(offset);
        
        uint8 guardianIndex;
        (guardianIndex, offset) = data.asUint8CdUnchecked(offset);
        
        bytes32[] memory tlsKeys = getTLSKeys(guardianSetIndex);
        
        result = abi.encodePacked(result, tlsKeys);
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