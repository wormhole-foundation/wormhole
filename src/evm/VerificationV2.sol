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
uint8 constant OP_THRESHOLD_GET_CURRENT = 0x22;
uint8 constant OP_THRESHOLD_GET = 0x23;
uint8 constant OP_GUARDIAN_SET_GET_CURRENT = 0x24;
uint8 constant OP_GUARDIAN_SET_GET = 0x25;
uint8 constant OP_GUARDIAN_SHARDS_GET = 0x26;

// Emitter address for the VerificationV2 contract
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

  error GuardianSetIsNotCurrent();

  error RegistrationMessageExpired();
  error GuardianSignatureVerificationFailed();

  /// @notice Emits an event when the owner successfully invalidates an unordered nonce.
  event UnorderedNonceInvalidation(address indexed owner, uint256 word, uint256 mask);

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
      (timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload) = _verifyAndDecodeThresholdVaa(data);
    } else if (version == 1) {
      (timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload) = _verifyAndDecodeMultisigVaa(data);
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
        ) = _verifyAndDecodeMultisigVaa(encodedVaa);

        // Verify the emitter
        require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
        require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

        // Get the guardian set
        (uint32 guardianSetIndex, address[] memory guardians) = _getCurrentGuardianSetInfo();

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
        uint32 thresholdKeyIndex;
        uint256 nonce;
        bytes32 guardianId;
        uint8 guardianIndex; bytes32 r; bytes32 s; uint8 v;

        (thresholdKeyIndex, offset) = data.asUint32CdUnchecked(offset);
        (nonce, offset) = data.asUint256CdUnchecked(offset);
        (guardianId, offset) = data.asBytes32CdUnchecked(offset);
        (guardianIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);

        // We only allow registrations for the current threshold key
        (ThresholdKeyInfo memory info, uint32 currentThresholdKeyIndex) = _getCurrentThresholdInfo();
        require(thresholdKeyIndex == currentThresholdKeyIndex, GuardianSetIsNotCurrent());

        // Replay protection
        _useUnorderedNonce(owner, nonce);

        // Get the guardian set for the threshold key
        uint32 guardianSetIndex = info.guardianSetIndex;
        (, address[] memory guardianAddrs) = _getGuardianSetInfo(guardianSetIndex);
        // TODO: Verify the guardian set is still valid? What about for the verification path?
        // We can't afford to check it there, so I'm skipping it here for now too

        // Verify the signature
        // We're not doing replay protection with the signature itself so we don't care about
        // verifying only canonical (low s) signatures.
        bytes32 digest = getRegisterGuardianDigest(thresholdKeyIndex, nonce, guardianId);
        address signatory = ecrecover(digest, v, r, s);
        require(signatory == guardianAddrs[guardianIndex], GuardianSignatureVerificationFailed());

        _registerGuardian(info, guardianIndex, guardianId);
      } else {
        revert InvalidOperation(op);
      }
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return new bytes(0);
  }

    /// @notice Invalidates the bits specified in mask for the bitmap at the word position
    /// @dev The wordPos is maxed at type(uint248).max
    /// @param wordPos A number to index the nonceBitmap at
    /// @param mask A bitmap masked against msg.sender's current bitmap at the word position
    function invalidateUnorderedNonces(uint256 wordPos, uint256 mask) external {
        nonceBitmap[msg.sender][wordPos] |= mask;

        emit UnorderedNonceInvalidation(msg.sender, wordPos, mask);
    }

    /// @notice Returns the index of the bitmap and the bit position within the bitmap. Used for unordered nonces
    /// @param nonce The nonce to get the associated word and bit positions
    /// @return wordPos The word position or index into the nonceBitmap
    /// @return bitPos The bit position
    /// @dev The first 248 bits of the nonce value is the index of the desired bitmap
    /// @dev The last 8 bits of the nonce value is the position of the bit in the bitmap
    function bitmapPositions(uint256 nonce) private pure returns (uint256 wordPos, uint256 bitPos) {
        wordPos = uint248(nonce >> 8);
        bitPos = uint8(nonce);
    }

    /// @notice Checks whether a nonce is taken and sets the bit at the bit position in the bitmap at the word position
    /// @param from The address to use the nonce at
    /// @param nonce The nonce to spend
    function _useUnorderedNonce(address from, uint256 nonce) internal {
        (uint256 wordPos, uint256 bitPos) = bitmapPositions(nonce);
        uint256 bit = 1 << bitPos;
        uint256 flipped = nonceBitmap[from][wordPos] ^= bit;

        if (flipped & bit == 0) revert InvalidNonce();
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
        ) = verifyAndDecodeVaa(encodedVaa);

        result = abi.encodePacked(
          result,
          abi.encode(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consistencyLevel,
            payload
          )
        );
      } else if (op == OP_VERIFY_VAA) {
        // Read the VAA
        bytes calldata encodedVaa;
        (encodedVaa, offset) = data.sliceUint16PrefixedCdUnchecked(offset);

        // Verify the VAA
        verifyVaa(encodedVaa);
      } else if (op == OP_THRESHOLD_GET_CURRENT) {
        (ThresholdKeyInfo memory info, uint32 index) = _getCurrentThresholdInfo();

        result = abi.encodePacked(result, abi.encode(info.pubkey, index));
      } else if (op == OP_THRESHOLD_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);

        ThresholdKeyInfo memory info = _getThresholdInfo(index);

        result = abi.encodePacked(result, abi.encode(info.pubkey, info.expirationTime));
      } else if (op == OP_GUARDIAN_SET_GET_CURRENT) {
        (uint32 guardianSet, address[] memory guardianSetAddrs) = _getCurrentGuardianSetInfo();

        result = abi.encodePacked(result, abi.encode(guardianSetAddrs, guardianSet));
      } else if (op == OP_GUARDIAN_SET_GET) {
        uint32 index;
        (index, offset) = data.asUint32CdUnchecked(offset);

        (uint32 expirationTime, address[] memory guardianSetAddrs) = _getGuardianSetInfo(index);

        result = abi.encodePacked(result, abi.encode(guardianSetAddrs, expirationTime));
      } else if (op == OP_GUARDIAN_SHARDS_GET) {
        uint32 guardianSet;
        (guardianSet, offset) = data.asUint32CdUnchecked(offset);

        ShardInfo[] memory shards = _getShards(guardianSet);

        result = abi.encodePacked(result, abi.encode(shards));
      } else {
        revert InvalidOperation(op);
      }
    }

    // Verify the data has been consumed
    data.length.checkLength(offset);

    return result;
  }
}
