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

// Emitter address for the VerificationV2 contract
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_THRESHOLD_KEY = 0x01;

contract VerificationV2 is 
  RawDispatcher, ThresholdVerification, GuardianSetVerification, GuardianRegistryVerification
{
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

  error InvalidDispatchVersion(uint8 version);
  error InvalidModule(bytes32 module);
  error InvalidAction(uint8 action);
  error InvalidValue();

  error InvalidGovernanceChainId();
  error InvalidGovernanceAddress();

  error GuardianSetIsNotCurrent();

  constructor(address coreV1, uint pullLimit) 
    ThresholdVerification()
    GuardianSetVerification(coreV1, pullLimit)
    GuardianRegistryVerification() 
  {}

  function _decodeAndVerifyVaa(bytes calldata encodedVaa) internal view returns (
    uint32  timestamp,
    uint32  nonce,
    uint16  emitterChainId,
    bytes32 emitterAddress,
    uint64  sequence,
    uint8   consistencyLevel,
    bytes calldata payload
  ) {
    (uint8 version, ) = encodedVaa.asUint8CdUnchecked(0);
    if (version == 2) {
      return verifyThresholdVAA(encodedVaa);
    } else if (version == 1) {
      return verifyGuardianSetVAA(encodedVaa);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function _decodeThresholdKeyUpdatePayload(bytes memory payload) internal pure returns (
    uint32 newThresholdIndex,
    address newThresholdAddr,
    uint32 expirationDelaySeconds,
    bytes32[] memory shards
  ) {
    // Decode and verify the module and action
    uint offset = 0;
    
    bytes32 module;
    (module, offset) = payload.asBytes32MemUnchecked(offset);
    if (module != MODULE_VERIFICATION_V2) revert InvalidModule(module);

    uint8 action;
    (action, offset) = payload.asUint8MemUnchecked(offset);
    if (action != ACTION_APPEND_THRESHOLD_KEY) revert InvalidAction(action);

    // Decode the message body
    (newThresholdIndex, offset) = payload.asUint32MemUnchecked(offset);
    (newThresholdAddr, offset) = payload.asAddressMemUnchecked(offset);
    (expirationDelaySeconds, offset) = payload.asUint32MemUnchecked(offset);

    uint8 shardsLength;
    (shardsLength, offset) = payload.asUint8MemUnchecked(offset);
    shards = new bytes32[](shardsLength);
    for (uint i = 0; i < shardsLength; ++i) {
      (shards[i], offset) = payload.asBytes32MemUnchecked(offset);
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
        (
          ,
          ,
          uint16 emitterChainId,
          bytes32 emitterAddress,
          ,
          ,
          bytes calldata payload
        ) = _decodeAndVerifyVaa(encodedVaa);

        // Verify the emitter
        if (emitterChainId != CHAIN_ID_SOLANA) revert InvalidGovernanceChainId();
        if (emitterAddress != GOVERNANCE_ADDRESS) revert InvalidGovernanceAddress();
        
        // Decode the payload
        (
          uint32 newThresholdIndex,
          address newThresholdAddr,
          uint32 expirationDelaySeconds,
          bytes32[] memory shards
        ) = _decodeThresholdKeyUpdatePayload(payload);
        
        // Append the threshold key
        _appendThresholdKey(newThresholdIndex, newThresholdAddr, expirationDelaySeconds, shards);
      } else if (op == OP_PULL_GUARDIAN_SETS) {
        uint32 limit;
        (limit, offset) = data.asUint32CdUnchecked(offset);

        pullGuardianSets(limit);
      } else if (op == OP_REGISTER_TLS_KEY) {
        uint32 guardianSetIndex;
        (guardianSetIndex, offset) = data.asUint32CdUnchecked(offset);
        uint32 expirationTime;
        (expirationTime, offset) = data.asUint32CdUnchecked(offset);
        bytes32 tlsKey;
        (tlsKey, offset) = data.asBytes32CdUnchecked(offset);
        uint8 guardianIndex; bytes32 r; bytes32 s; uint8 v;
        (guardianIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);
        // We only allow registrations for the current guardian set
        (uint32 currentSetIndex, address[] memory guardianAddrs) = this.getCurrentGuardianSetInfo();
        if (guardianSetIndex != currentSetIndex) revert GuardianSetIsNotCurrent();
        verifyRegisterTLSKey(
          guardianAddrs,
          guardianSetIndex,
          expirationTime,
          tlsKey,
          guardianIndex,
          r, s, v
        );
        registerTLSKey(guardianSetIndex, guardianIndex, tlsKey, guardianAddrs.length);
      }
      // TODO: Isn't this missing an else revert?
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
      
      if (op == OP_VERIFY) {
        // Read the VAA
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        // Decode the VAA
        (
          uint32 timestamp,
          uint32 nonce,
          uint16 emitterChainId,
          bytes32 emitterAddress,
          uint64 sequence,
          uint8 consistencyLevel,
          bytes calldata payload
        ) = _decodeAndVerifyVaa(encodedVaa);

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
      }
      // TODO: Isn't this missing an else revert?
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return result;
  }
}
