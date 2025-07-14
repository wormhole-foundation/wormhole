// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerAnd} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import {RawDispatcher} from "wormhole-sdk/RawDispatcher.sol";
import {CHAIN_ID_SOLANA} from "wormhole-sdk/constants/Chains.sol";
import {ICoreBridge, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";

import {ThresholdVerification, ShardInfo} from "./ThresholdVerification.sol";
import {MultisigVerification} from "./MultisigVerification.sol";
import {EIP712Encoding} from "./EIP712Encoding.sol";

// Raw dispatch operation IDs for exec
uint8 constant OP_APPEND_THRESHOLD_KEY = 0x00;
uint8 constant OP_PULL_GUARDIAN_SETS = 0x01;
uint8 constant OP_REGISTER_GUARDIAN = 0x02;

// Raw dispatch operation IDs for get
uint8 constant OP_VERIFY_AND_DECODE_VAA = 0x20;
uint8 constant OP_VERIFY_VAA = 0x21;
uint8 constant OP_GET_THRESHOLD_CURRENT = 0x22;
uint8 constant OP_GET_THRESHOLD = 0x23;
uint8 constant OP_GET_GUARDIAN_SET_CURRENT = 0x24;
uint8 constant OP_GET_GUARDIAN_SET = 0x25;
uint8 constant OP_GET_SHARDS = 0x26;

// Governance emitter address
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

contract VerificationV2 is
  RawDispatcher, ThresholdVerification, MultisigVerification, EIP712Encoding
{
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using {BytesParsing.checkLength} for uint;

  error InvalidValue();
  error InvalidOperation(uint8 op);
  error InvalidGovernanceChain();
  error InvalidGovernanceAddress();

  error ThresholdKeyIsNotCurrent();

  error RegistrationMessageExpired();
  error GuardianSignatureVerificationFailed();
  error InvalidNonce();

  mapping(address => mapping(uint256 => uint256)) public nonceBitmap;

  constructor(ICoreBridge coreV1, uint256 initGuardianSetIndex, uint256 pullLimit)
    MultisigVerification(coreV1, initGuardianSetIndex, pullLimit)
  {}

  function verifyVaa(bytes calldata data) public view {
    (uint8 version, ) = data.asUint8CdUnchecked(0);
    if (version == 2) {
      _verifyThresholdVaaHeader(data);
    } else if (version == 1) {
      _verifyMultisigVaaHeader(data);
    } else {
      revert VaaLib.InvalidVersion(version);
    }
  }

  function verifyAndDecodeVaa(bytes calldata data) public view returns (
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
      return _verifyAndDecodeThresholdVaa(data);
    } else if (version == 1) {
      return _verifyAndDecodeMultisigVaa(data);
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

      if (op == OP_APPEND_THRESHOLD_KEY) {
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        (
          ,
          ,
          uint16 emitterChainId,
          bytes32 emitterAddress,
          ,
          ,
          bytes calldata payload
        ) = _verifyAndDecodeMultisigVaa(encodedVaa);

        require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
        require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

        (uint32 guardianSetIndex, address[] memory guardians) = _getCurrentGuardianSetInfo();

        (
          uint32 newTSSIndex,
          uint256 newThresholdAddr,
          uint32 expirationDelaySeconds,
          ShardInfo[] memory shards
        ) = _decodeThresholdKeyUpdatePayload(payload, guardians.length);

        _appendThresholdKey(guardianSetIndex, newTSSIndex, newThresholdAddr, expirationDelaySeconds, shards);
      } else if (op == OP_PULL_GUARDIAN_SETS) {
        uint32 limit;
        (limit, offset) = data.asUint32CdUnchecked(offset);

        _pullGuardianSets(limit);
      } else if (op == OP_REGISTER_GUARDIAN) {
        // Decode the payload
        uint32 thresholdKeyIndex;
        uint256 nonce;
        bytes32 guardianId;
        uint8 guardianIndex; bytes32 r; bytes32 s; uint8 v;

        (thresholdKeyIndex, offset) = data.asUint32CdUnchecked(offset);
        (nonce, offset) = data.asUint256CdUnchecked(offset);
        (guardianId, offset) = data.asBytes32CdUnchecked(offset);
        (guardianIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);

        (ThresholdKeyInfo memory info, uint32 currentThresholdKeyIndex) = _getCurrentThresholdInfo();
        require(thresholdKeyIndex == currentThresholdKeyIndex, ThresholdKeyIsNotCurrent());

        uint32 guardianSetIndex = info.guardianSetIndex;
        (, address[] memory guardianAddrs) = _getGuardianSetInfo(guardianSetIndex);

        // We're not doing replay protection with the signature itself so we don't care about
        // verifying only canonical (low s) signatures.
        bytes32 digest = getRegisterGuardianDigest(thresholdKeyIndex, nonce, guardianId);
        address signatory = ecrecover(digest, v, r, s);
        require(signatory == guardianAddrs[guardianIndex], GuardianSignatureVerificationFailed());

        _useUnorderedNonce(signatory, nonce);

        _registerGuardian(info, guardianIndex, guardianId);
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
        ) = verifyAndDecodeVaa(encodedVaa);

        result = abi.encodePacked(
          result,
          timestamp,
          nonce,
          emitterChainId,
          emitterAddress,
          sequence,
          consistencyLevel,
          uint16(payload.length),
          payload
        );
      } else if (op == OP_VERIFY_VAA) {
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        verifyVaa(encodedVaa);
      } else if (op == OP_GET_THRESHOLD_CURRENT) {
        (ThresholdKeyInfo memory info, uint32 index) = _getCurrentThresholdInfo();

        result = abi.encodePacked(result, info.pubkey, index);
      } else if (op == OP_GET_THRESHOLD) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);

        ThresholdKeyInfo memory info = _getThresholdInfo(index);

        result = abi.encodePacked(result, info.pubkey, info.expirationTime);
      } else if (op == OP_GET_GUARDIAN_SET_CURRENT) {
        (uint32 guardianSetIndex, address[] memory guardianSetAddrs) = _getCurrentGuardianSetInfo();

        result = abi.encodePacked(result, uint8(guardianSetAddrs.length), guardianSetAddrs, guardianSetIndex);
      } else if (op == OP_GET_GUARDIAN_SET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);

        (uint32 expirationTime, address[] memory guardianSetAddrs) = _getGuardianSetInfo(index);
        result = abi.encodePacked(result, uint8(guardianSetAddrs.length), guardianSetAddrs, expirationTime);
      } else if (op == OP_GET_SHARDS) {
        uint32 guardianSet;
        (guardianSet, offset) = data.asUint32CdUnchecked(offset);

        bytes memory rawShards = _getShardsRaw(guardianSet);
        result = abi.encodePacked(result, rawShards);
      } else {
        revert InvalidOperation(op);
      }
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return result;
  }

  /// @notice Checks whether a nonce is taken and sets the bit at the bit position in the bitmap at the word position
  /// @param nonce The nonce to spend
  function _useUnorderedNonce(address guardian, uint256 nonce) internal {
    uint256 wordPos = uint248(nonce >> 8);
    uint256 bitPos = uint8(nonce);
    uint256 bit = 1 << bitPos;
    uint256 flipped = nonceBitmap[guardian][wordPos] ^= bit;

    if (flipped & bit == 0) revert InvalidNonce();
  }
}
