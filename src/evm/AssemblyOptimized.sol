// SPDX-License-Identifier: MIT

pragma solidity ^0.8.28;

import {console} from "forge-std/console.sol";

import {RawDispatcher} from "wormhole-solidity-sdk/RawDispatcher.sol";
import {eagerAnd} from "wormhole-solidity-sdk/Utils.sol";
import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {ICoreBridge, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {VaaLib, VaaBody} from "wormhole-solidity-sdk/libraries/VaaLib.sol";

import {EIP712Encoding} from "./EIP712Encoding.sol";

struct ShardData {
	bytes32 shard;
	bytes32 id;
}

// Governance emitter address
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_SCHNORR_KEY = 0x01;

uint8 constant EXEC_UPDATE_SHARD_ID = 0;
uint8 constant EXEC_APPEND_SCHNORR_KEY = 1;
uint8 constant EXEC_PULL_MULTISIG_KEY_DATA = 2;

uint8 constant GET_CURRENT_SCHNORR_KEY_DATA = 0;
uint8 constant GET_CURRENT_MULTISIG_KEY_DATA = 1;
uint8 constant GET_SCHNORR_KEY_DATA = 2;
uint8 constant GET_MULTISIG_KEY_DATA = 3;
uint8 constant GET_SCHNORR_SHARD_DATA = 4;

contract VerificationCore {
  uint256 private constant SLOT_CORE_BRIDGE = 1000;
  uint256 private constant SLOT_MULTISIG_KEY_COUNT = 1001;
  uint256 private constant SLOT_SCHNORR_KEY_COUNT = 1002;
  uint256 private constant SLOT_SCHNORR_SHARD_COUNT = 1003;

  uint256 private constant SLOT_MULTISIG_KEY_DATA = 1 << 48;
  uint256 private constant SLOT_SCHNORR_KEY_DATA = 2 << 48;
  uint256 private constant SLOT_SCHNORR_KEY_EXTRA = 3 << 48;
  uint256 private constant SLOT_SCHNORR_SHARD_DATA = 4 << 48;

  uint256 private constant MASK_MULTISIG_ENTRY_EXPIRATION_TIME = 0xFFFFFFFF;
  uint256 private constant SHIFT_MULTISIG_ENTRY_ADDRESS = 32;

  uint256 private constant OFFSET_MULTISIG_CONTRACT_DATA = 1;

  uint256 private constant MASK_SCHNORR_EXTRA_EXPIRATION_TIME = 0xFFFFFFFF;
  uint256 private constant SHIFT_SCHNORR_EXTRA_SHARD_COUNT = 32;
  uint256 private constant MASK_SCHNORR_EXTRA_SHARD_COUNT = 0xFF;
  uint256 private constant SHIFT_SCHNORR_EXTRA_SHARD_BASE = 40;
  uint256 private constant MASK_SCHNORR_EXTRA_SHARD_BASE = 0xFFFFFFFFFF;
  uint256 private constant SHIFT_SCHNORR_EXTRA_MULTISIG_KEY_INDEX = 80;
  uint256 private constant MASK_SCHNORR_EXTRA_MULTISIG_KEY_INDEX = 0xFFFFFFFF;

  uint256 private constant VAA_MULTISIG_SIGNATURE_COUNT_OFFSET = 5;
  uint256 private constant VAA_MULTISIG_SIGNATURE_OFFSET = 6;
  uint256 private constant VAA_MULTISIG_SIGNATURE_R_OFFSET = 1;
  uint256 private constant VAA_MULTISIG_SIGNATURE_S_OFFSET = 33;
  uint256 private constant VAA_MULTISIG_SIGNATURE_V_OFFSET = 65;
  uint256 private constant VAA_MULTISIG_SIGNATURE_SIZE = 66;

	// Curve order for secp256k1
  uint256 constant internal Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant internal HALF_Q = 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0; // Q >> 1

  error InvalidOpcode(uint256 offset); // 0x6ecdda1e
  error DeploymentFailed(); // 0x30116425
  error InvalidPointer(); // 0x11052bb4
  error KeyDataExpired(); // 0xf8375bbc
  error VerificationFailed(); // 0x439cc0cd
  error InvalidKey(); // 0x76d4e1e8

  constructor(
    ICoreBridge coreBridge,
    uint32 initialMultisigKeyCount,
    uint32 initialSchnorrKeyCount
  ) {
    assembly ("memory-safe") {
      sstore(SLOT_CORE_BRIDGE, coreBridge)
      sstore(SLOT_MULTISIG_KEY_COUNT, initialMultisigKeyCount)
      sstore(SLOT_SCHNORR_KEY_COUNT, initialSchnorrKeyCount)
    }
  }
  
  function _getCoreBridge() internal view returns (ICoreBridge result) {
    assembly ("memory-safe") {
      result := sload(SLOT_CORE_BRIDGE)
    }
  }

  function _getMultisigKeyCount() internal view returns (uint256 result) {
    assembly ("memory-safe") {
      result := sload(SLOT_MULTISIG_KEY_COUNT)
    }
  }

  function _getMultisigKeyData(uint256 index) internal view returns (uint8 keyCount, uint256 keyDataOffset, uint32 expirationTime) {
    assembly ("memory-safe") {
      // Load and decode the multisig key data entry
      let entry := sload(add(SLOT_MULTISIG_KEY_DATA, index))
      expirationTime := and(entry, MASK_MULTISIG_ENTRY_EXPIRATION_TIME)
      let keyDataAddress := shr(SHIFT_MULTISIG_ENTRY_ADDRESS, entry)
      // Load the key data contract, validate the size
      let keyDataSize := extcodesize(keyDataAddress)
      if iszero(keyDataSize) {
        mstore(0, 0x11052bb4) // InvalidPointer()
        revert(0, 0x04)
      }

      // Copy the value to memory
      let size := sub(keyDataSize, OFFSET_MULTISIG_CONTRACT_DATA)
      keyCount := shr(5, size)

      keyDataOffset := mload(0x40)
      mstore(0x40, add(keyDataOffset, size))
      extcodecopy(keyDataAddress, keyDataOffset, OFFSET_MULTISIG_CONTRACT_DATA, size)
    }
  }

  function _setMultisigExpirationTime(uint32 index, uint32 expirationTime) internal {
    assembly ("memory-safe") {
      let ptr := add(SLOT_MULTISIG_KEY_DATA, index)
      sstore(ptr, or(and(sload(ptr), MASK_MULTISIG_ENTRY_EXPIRATION_TIME), expirationTime))
    }
  }

  function _appendMultisigKeyData(address[] memory keys, uint32 expirationTime) internal {
    assembly ("memory-safe") {
      // Deploy the data to a new contract
      let originalDataLength := shl(5, mload(keys))
      let dataLength := add(originalDataLength, OFFSET_MULTISIG_CONTRACT_DATA)
      mstore(add(keys, gt(dataLength, 0xFFFF)), or(0xfd61000080600a3d393df300, shl(0x40, dataLength)))
      let deployedAddress := create(0, add(keys, 0x15), add(dataLength, 0xA))
      if iszero(deployedAddress) {
        mstore(0, 0x30116425) // DeploymentFailed()
        revert(0, 0x04)
      }

      // Restore the original length of the variable size `keys`
      mstore(keys, originalDataLength)

      // Store the entry in the storage array
      let index := sload(SLOT_MULTISIG_KEY_COUNT)
      let entry := or(expirationTime, shl(SHIFT_MULTISIG_ENTRY_ADDRESS, deployedAddress))
      sstore(add(SLOT_MULTISIG_KEY_DATA, index), entry)

      // Increment the multisig key count
      sstore(SLOT_MULTISIG_KEY_COUNT, add(index, 1))
    }
  }

  function _getSchnorrKeyCount() internal view returns (uint256 result) {
    assembly ("memory-safe") {
      result := sload(SLOT_SCHNORR_KEY_COUNT)
    }
  }

  function _getSchnorrPubkey(uint256 index) internal view returns (uint256 result) {
    assembly ("memory-safe") {
      result := sload(add(SLOT_SCHNORR_KEY_DATA, index))
    }
  }

  function _getSchnorrExtraInfo(uint256 index) internal view returns (uint32 expirationTime, uint8 shardCount, uint40 shardBase, uint32 multisigKeyIndex) {
    assembly ("memory-safe") {
      let result := sload(add(SLOT_SCHNORR_KEY_EXTRA, index))
      expirationTime := and(result, MASK_SCHNORR_EXTRA_EXPIRATION_TIME)
      shardCount := and(shr(SHIFT_SCHNORR_EXTRA_SHARD_COUNT, result), MASK_SCHNORR_EXTRA_SHARD_COUNT)
      shardBase := and(shr(SHIFT_SCHNORR_EXTRA_SHARD_BASE, result), MASK_SCHNORR_EXTRA_SHARD_BASE)
      multisigKeyIndex := shr(SHIFT_SCHNORR_EXTRA_MULTISIG_KEY_INDEX, result)
    }
  }

  function _setSchnorrExpirationTime(uint32 index, uint32 expirationTime) internal {
    assembly ("memory-safe") {
      let ptr := add(SLOT_SCHNORR_KEY_EXTRA, index)
      sstore(ptr, or(and(sload(ptr), MASK_SCHNORR_EXTRA_EXPIRATION_TIME), expirationTime))
    }
  }

  function _getSchnorrRawShardData(uint32 index) internal view returns (bytes memory result) {
    (, uint8 shardCount, uint40 shardBase,) = _getSchnorrExtraInfo(index);

    assembly ("memory-safe") {
      result := mload(0x40)
      let shardBytes := shl(6, shardCount)
      mstore(0x40, add(result, add(0x20, shardBytes)))
      mstore(result, shardBytes)

      let readPtr := add(SLOT_SCHNORR_SHARD_DATA, shardBase)
      let writePtr := add(0x20, result)
      for {let i := 0} lt(i, shardCount) {i := add(i, 1)} {
        mstore(writePtr, sload(readPtr))
        mstore(add(writePtr, 0x20), sload(add(readPtr, 1)))
        writePtr := add(writePtr, 0x40)
        readPtr := add(readPtr, 2)
      }
    }
  }

  function _setSchnorrShardId(uint40 shardBase, uint8 shardIndex, bytes32 shardId) internal {
    assembly ("memory-safe") {
      let ptr := add(add(shardBase, 1), shl(1, shardIndex))
      sstore(ptr, shardId)
    }
  }

  function _appendSchnorrKeyData(
    uint256 pubkey,
    uint32 multisigKeyIndex,
    uint8 shardCount,
    bytes calldata shardData,
    uint256 offset
  ) internal returns (uint256 newOffset) {
    assembly ("memory-safe") {
      // Validate the pubkey
      let px := shr(1, pubkey)
      if or(iszero(px), gt(px, HALF_Q)) {
        mstore(0, 0x76d4e1e8) // InvalidKey()
        revert(0, 0x04)
      }

      // Append the key data
      let keyIndex := sload(SLOT_SCHNORR_KEY_COUNT)
      sstore(add(SLOT_SCHNORR_KEY_DATA, keyIndex), pubkey)

      let shardBase := sload(SLOT_SCHNORR_SHARD_COUNT)
      let extraInfo := or(
        shl(SHIFT_SCHNORR_EXTRA_SHARD_COUNT, shardCount),
        or(
          shl(SHIFT_SCHNORR_EXTRA_SHARD_BASE, shardBase),
          shl(SHIFT_SCHNORR_EXTRA_MULTISIG_KEY_INDEX, multisigKeyIndex)
        )
      )

      sstore(add(SLOT_SCHNORR_KEY_EXTRA, keyIndex), extraInfo)

      // Append the shard data
      let readPtr := add(shardData.offset, offset)
      let writePtr := add(SLOT_SCHNORR_SHARD_DATA, shardBase)
      for {let i := 0} lt(i, shardCount) {i := add(i, 1)} {
        sstore(writePtr, calldataload(readPtr))
        sstore(add(writePtr, 1), calldataload(add(readPtr, 0x20)))
        writePtr := add(writePtr, 2)
        readPtr := add(readPtr, 0x40)
      }

      // Update the lengths
      sstore(SLOT_SCHNORR_KEY_COUNT, add(keyIndex, 1))
      sstore(SLOT_SCHNORR_SHARD_COUNT, add(shardBase, shl(1, shardCount)))

      // Return the new offset
      newOffset := sub(readPtr, shardData.offset)
    }
  }

  function _verifyVaa(bytes calldata data) internal view returns (uint256 envelopeOffset) {
    assembly ("memory-safe") {
      let version := shr(248, calldataload(data.offset))
      let keyIndex := shr(224, calldataload(add(data.offset, 1)))
      let buffer := mload(0x40)
      
      switch version
      case 2 {
        // Decode the signature
        envelopeOffset := 57
        let r := shr(96, calldataload(add(data.offset, 5)))
        let s := calldataload(add(data.offset, 25))

        // Compute the double hash of the VAA
        let envelopeLength := sub(data.length, envelopeOffset)
        calldatacopy(buffer, add(data.offset, envelopeOffset), envelopeLength)
        let singleHash := keccak256(buffer, envelopeLength)
        mstore(0, singleHash)
        let digest := keccak256(0, 32)

        // Load the key and validate the expiration time
        let pubkey := sload(add(SLOT_SCHNORR_KEY_DATA, keyIndex))
        let expirationTime := and(sload(add(SLOT_SCHNORR_KEY_EXTRA, keyIndex)), 0xFFFFFFFF)

        // NOTE: Inline from _verifySchnorr to save gas
        // Compute the challenge value
        let px := shr(1, pubkey)
        let parity := and(pubkey, 1)
        mstore(buffer, px)
        mstore8(add(buffer, 32), parity)
        mstore(add(buffer, 33), digest)
        mstore(add(buffer, 65), shl(96, r))
        let e := keccak256(buffer, 85)

        // Call ecrecover
        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
        mstore(buffer, sub(Q, mulmod(px, s, Q)))
        mstore(add(buffer, 32), add(parity, 27))
        mstore(add(buffer, 64), px)
        mstore(add(buffer, 96), mulmod(px, e, Q))
        let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

        // Validate the result
        let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))
        let validPubkey := not(iszero(px))
        let validSignature := and(not(iszero(r)), lt(s, Q))
        let recoveredValid := and(success, eq(r, mload(buffer)))
        let valid := and(expirationTimeValid, and(validPubkey, and(validSignature, recoveredValid)))

        envelopeOffset := mul(envelopeOffset, valid)
      }
      case 1 {
        // Decode the signature count
        let signatureCount := shr(248, calldataload(add(data.offset, VAA_MULTISIG_SIGNATURE_COUNT_OFFSET)))
        envelopeOffset := add(VAA_MULTISIG_SIGNATURE_OFFSET, mul(signatureCount, VAA_MULTISIG_SIGNATURE_SIZE))

        // Compute the double hash of the VAA
        let envelopeLength := sub(data.length, envelopeOffset)
        calldatacopy(buffer, add(data.offset, envelopeOffset), envelopeLength)
        let singleHash := keccak256(buffer, envelopeLength)
        mstore(0, singleHash)
        let digest := keccak256(0, 32)

        // NOTE: Inline from _getMultisigKeyData to save gas
        // Load the key data and validate the expiration time
        let entry := sload(add(SLOT_MULTISIG_KEY_DATA, keyIndex))
        let expirationTime := and(entry, MASK_MULTISIG_ENTRY_EXPIRATION_TIME)
        let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))
        let keyDataAddress := shr(SHIFT_MULTISIG_ENTRY_ADDRESS, entry)

        // Load the key data contract, validate the size
        let keyDataSize := extcodesize(keyDataAddress)
        let keyDataSizeValid := not(iszero(keyDataSize))

        // Copy the value to memory
        let size := sub(keyDataSize, OFFSET_MULTISIG_CONTRACT_DATA)
        let keyCount := shr(5, size)

        let keyDataOffset := buffer
        buffer := add(buffer, size)
        extcodecopy(keyDataAddress, keyDataOffset, OFFSET_MULTISIG_CONTRACT_DATA, size)

        // Verify the quorum
        let quorum := div(shl(1, keyCount), 3)
        let quorumValid := gt(signatureCount, quorum)

        // NOTE: Inline from _verifyMultisig to save gas
        // Verify the signatures
        let usedSignerBitfield := 0
        let valid := and(expirationTimeValid, and(keyDataSizeValid, quorumValid))

        let ptr := add(data.offset, VAA_MULTISIG_SIGNATURE_OFFSET)
        for {let i := 0} lt(i, signatureCount) {i := add(i, 1)} {
          let signerIndex := shr(248, calldataload(ptr))
          let r := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_R_OFFSET))
          let s := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_S_OFFSET))
          let v := shr(248, calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_V_OFFSET)))

          // Call ecrecover
          mstore(buffer, digest)
          mstore(add(buffer, 32), add(v, 27))
          mstore(add(buffer, 64), r)
          mstore(add(buffer, 96), s)
          let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

          // Validate the result
          let recovered := mload(buffer)
          let expected := mload(add(keyDataOffset, shl(5, signerIndex)))
          let signatureValid := eq(expected, recovered)
          let indexValid := lt(signerIndex, keyCount)
          let signerFlag := shl(signerIndex, 1)
          let signerUsedValid := iszero(and(usedSignerBitfield, signerFlag))

          valid := and(valid, and(and(indexValid, signatureValid), signerUsedValid))
          usedSignerBitfield := or(usedSignerBitfield, signerFlag)
          ptr := add(ptr, VAA_MULTISIG_SIGNATURE_SIZE)
        }

        envelopeOffset := mul(envelopeOffset, valid)
      }
      default {
        envelopeOffset := 0
      }
    }
  }

  function _verifyMultisig(bytes32 digest, uint8 keyCount, uint256 keyDataOffset, uint8 signatureCount, bytes calldata signatures, uint256 signaturesOffset) internal view returns (bool valid) {
    assembly ("memory-safe") {
      let usedSignerBitfield := 0
      valid := 1

      let buffer := mload(0x40)
      let ptr := add(signatures.offset, signaturesOffset)
      for {let i := 0} lt(i, signatureCount) {i := add(i, 1)} {
        let signerIndex := shr(248, calldataload(ptr))
        let r := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_R_OFFSET))
        let s := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_S_OFFSET))
        let v := shr(248, calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_V_OFFSET)))

        // Call ecrecover
        mstore(buffer, digest)
        mstore(add(buffer, 32), add(v, 27))
        mstore(add(buffer, 64), r)
        mstore(add(buffer, 96), s)
        let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

        // Validate the result
        let recovered := mload(buffer)
        let expected := mload(add(keyDataOffset, shl(5, signerIndex)))
        let signatureValid := eq(expected, recovered)
        let indexValid := lt(signerIndex, keyCount)
        let signerFlag := shl(signerIndex, 1)
        let signerUsedValid := iszero(and(usedSignerBitfield, signerFlag))

        valid := and(valid, and(and(indexValid, signatureValid), signerUsedValid))
        usedSignerBitfield := or(usedSignerBitfield, signerFlag)
        ptr := add(ptr, VAA_MULTISIG_SIGNATURE_SIZE)
      }
    }
  }

  function _verifySchnorr(bytes32 digest, uint256 pubkey, address r, uint256 s) internal view returns (bool valid) {
    assembly ("memory-safe") {
      // Compute the challenge value
      let px := shr(1, pubkey)
      let parity := and(pubkey, 1)
      let buffer := mload(0x40)
      mstore(buffer, px)
      mstore8(add(buffer, 32), parity)
      mstore(add(buffer, 33), digest)
      mstore(add(buffer, 65), shl(96, r))
      let e := keccak256(buffer, 85)

      // Call ecrecover
      // NOTE: This is non-zero because for all k = px * s, Q > k % Q
      //       Therefore, Q - k % Q is always positive
      mstore(buffer, sub(Q, mulmod(px, s, Q)))
      mstore(add(buffer, 32), add(parity, 27))
      mstore(add(buffer, 64), px)
      mstore(add(buffer, 96), mulmod(px, e, Q))
      let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

      // Validate the result
      let validPubkey := not(iszero(px))
      let validSignature := and(not(iszero(r)), lt(s, Q))
      let recoveredValid := and(success, eq(r, mload(buffer)))
      valid := and(validPubkey, and(validSignature, recoveredValid))
    }
  }

  function _verifyHashAndHeader(bytes calldata data) internal view {
    assembly ("memory-safe") {
      let digest := calldataload(data.offset)
      let version := shr(248, calldataload(add(data.offset, 32)))
      let keyIndex := shr(224, calldataload(add(data.offset, 33)))
      let buffer := mload(0x40)

      switch version
      case 2 {
        let r := shr(96, calldataload(add(data.offset, 37)))
        let s := calldataload(add(data.offset, 57))

        let pubkey := sload(add(SLOT_SCHNORR_KEY_DATA, keyIndex))
        let px := shr(1, pubkey)
        let parity := and(pubkey, 1)
        let pubkeyValid := not(iszero(px))

        let expirationTime := and(sload(add(SLOT_SCHNORR_KEY_EXTRA, keyIndex)), 0xFFFFFFFF)
        let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))

        // Inline from _verifySchnorr to save gas
        // Compute the challenge value
        mstore(buffer, px)
        mstore8(add(buffer, 32), parity)
        mstore(add(buffer, 33), digest)
        mstore(add(buffer, 65), shl(96, r))
        let e := keccak256(buffer, 85)

        // Call ecrecover
        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
        mstore(buffer, sub(Q, mulmod(px, s, Q)))
        mstore(add(buffer, 32), add(parity, 27))
        mstore(add(buffer, 64), px)
        mstore(add(buffer, 96), mulmod(px, e, Q))
        let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

        // Validate the result
        let recoveredValid := and(success, eq(r, mload(buffer)))
        let valid := and(pubkeyValid, and(expirationTimeValid, recoveredValid))
        if iszero(valid) {
          mstore(0, 0x439cc0cd) // VerificationFailed()
          revert(0, 0x04)
        }
      }
      case 1 {
        let signatureCount := shr(248, calldataload(add(data.offset, 37)))

        // NOTE: Inline from _getMultisigKeyData to save gas
        // Load the key data and validate the expiration time
        let entry := sload(add(SLOT_MULTISIG_KEY_DATA, keyIndex))
        let expirationTime := and(entry, MASK_MULTISIG_ENTRY_EXPIRATION_TIME)
        let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))
        let keyDataAddress := shr(SHIFT_MULTISIG_ENTRY_ADDRESS, entry)

        // Load the key data contract, validate the size
        let keyDataSize := extcodesize(keyDataAddress)
        let keyDataSizeValid := not(iszero(keyDataSize))

        // Copy the value to memory
        let size := sub(keyDataSize, OFFSET_MULTISIG_CONTRACT_DATA)
        let keyCount := shr(5, size)

        let keyDataOffset := buffer
        buffer := add(buffer, size)
        extcodecopy(keyDataAddress, keyDataOffset, OFFSET_MULTISIG_CONTRACT_DATA, size)

        // Verify the quorum
        let quorum := div(shl(1, keyCount), 3)
        let quorumValid := gt(signatureCount, quorum)

        // NOTE: Inline from _verifyMultisig to save gas
        // Verify the signatures
        let usedSignerBitfield := 0
        let valid := and(expirationTimeValid, and(keyDataSizeValid, quorumValid))

        let ptr := add(data.offset, VAA_MULTISIG_SIGNATURE_OFFSET)
        for {let i := 0} lt(i, signatureCount) {i := add(i, 1)} {
          let signerIndex := shr(248, calldataload(ptr))
          let r := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_R_OFFSET))
          let s := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_S_OFFSET))
          let v := shr(248, calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_V_OFFSET)))

          // Call ecrecover
          mstore(buffer, digest)
          mstore(add(buffer, 32), add(v, 27))
          mstore(add(buffer, 64), r)
          mstore(add(buffer, 96), s)
          let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

          // Validate the result
          let recovered := mload(buffer)
          let expected := mload(add(keyDataOffset, shl(5, signerIndex)))
          let signatureValid := eq(expected, recovered)
          let indexValid := lt(signerIndex, keyCount)
          let signerFlag := shl(signerIndex, 1)
          let signerUsedValid := iszero(and(usedSignerBitfield, signerFlag))

          valid := and(valid, and(and(indexValid, signatureValid), signerUsedValid))
          usedSignerBitfield := or(usedSignerBitfield, signerFlag)
          ptr := add(ptr, VAA_MULTISIG_SIGNATURE_SIZE)
        }

        if iszero(valid) {
          mstore(0, 0x439cc0cd) // VerificationFailed()
          revert(0, 0x04)
        }
      }
      default {
        mstore(buffer, 0x7207be20) // InvalidVersion(uint8)
        mstore(add(buffer, 4), version)
        revert(buffer, 0x24)
      }
    }
  }
}

contract Verification is RawDispatcher, VerificationCore, EIP712Encoding {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error InvalidMultisigKeyIndex();
  error InvalidSignatureCount();
  error InvalidGovernanceChain();
  error InvalidGovernanceAddress();
  error InvalidModule();
  error InvalidAction();
  error InvalidKeyIndex();
  error InvalidDataLength();

  error SignatureExpired();
  error InvalidSignerIndex();

  constructor(
    ICoreBridge coreBridge,
    uint32 initialMultisigKeyCount,
    uint32 initialSchnorrKeyCount,
    uint32 initialMultisigKeyPullLimit
  ) VerificationCore(coreBridge, initialMultisigKeyCount, initialSchnorrKeyCount) {
    _pullMultisigKeyData(initialMultisigKeyPullLimit);
  }

  function verifyVaa_U7N5(bytes calldata data) external view {
    uint256 envelopeOffset = _verifyVaa(data);
    require(envelopeOffset != 0, VerificationFailed());
  }

  function verifyVaaDecodeEssentials(bytes calldata data) external view returns (
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    bytes memory payload
  ) {
    unchecked {
      uint256 envelopeOffset = _verifyVaa(data);

      uint256 offset = envelopeOffset + VaaLib.ENVELOPE_EMITTER_CHAIN_ID_OFFSET;
      (emitterChainId, offset) = data.asUint16CdUnchecked(offset);
      (emitterAddress, offset) = data.asBytes32CdUnchecked(offset);
      (sequence,             ) = data.asUint32CdUnchecked(offset);

      uint payloadOffset = envelopeOffset + VaaLib.ENVELOPE_SIZE;
      payload = data.decodeVaaPayloadCd(payloadOffset);
    }
  }

  function verifyVaaBody(bytes calldata data) external view returns (VaaBody memory body) {
    uint256 envelopeOffset = _verifyVaa(data);
    return data.decodeVaaBodyStructCd(envelopeOffset);
  }

  function verifyHashAndHeader(bytes calldata data) external view {
    _verifyHashAndHeader(data);
  }

  function _exec(bytes calldata data) internal override returns (bytes memory) {
    unchecked {
      uint256 offset = 0;

      while (offset < data.length) {
        uint8 opcode;

        (opcode, offset) = data.asUint8CdUnchecked(offset);

        if (opcode == EXEC_UPDATE_SHARD_ID) {
          offset = _updateShardId(data, offset);
        } else if (opcode == EXEC_APPEND_SCHNORR_KEY) {
          offset = _appendSchnorrKeys(data, offset);
        } else if (opcode == EXEC_PULL_MULTISIG_KEY_DATA) {
          uint32 limit;
          (limit, offset) = data.asUint32CdUnchecked(offset);
          _pullMultisigKeyData(limit);
        } else {
          revert InvalidOpcode(offset);
        }
      }

      require(offset == data.length, InvalidDataLength());
      return new bytes(0);
    }
  }

  function _get(bytes calldata data) internal view override returns (bytes memory) {
    unchecked {
      uint256 offset = 0;
      bytes memory result;

      while (offset < data.length) {
        uint8 opcode;

        (opcode, offset) = data.asUint8CdUnchecked(offset);

        if (opcode == GET_CURRENT_SCHNORR_KEY_DATA) {
          uint256 index = _getSchnorrKeyCount() - 1;
          uint256 pubkey = _getSchnorrPubkey(index);
          (uint32 expirationTime, , , uint32 multisigKeyIndex) = _getSchnorrExtraInfo(index);

          // TODO: Do we want to include the shard count here?
          result = abi.encodePacked(result, index, pubkey, expirationTime, multisigKeyIndex);
        } else if (opcode == GET_CURRENT_MULTISIG_KEY_DATA) {
        } else if (opcode == GET_SCHNORR_KEY_DATA) {
          uint32 index;
          (index, offset) = data.asUint32CdUnchecked(offset);

          uint256 pubkey = _getSchnorrPubkey(index);
          (uint32 expirationTime, uint8 shardCount, , uint32 multisigKeyIndex) = _getSchnorrExtraInfo(index);

          result = abi.encodePacked(result, pubkey, expirationTime, shardCount, multisigKeyIndex);
        } else if (opcode == GET_MULTISIG_KEY_DATA) {
        } else if (opcode == GET_SCHNORR_SHARD_DATA) {
          uint32 schnorrKeyIndex;
          (schnorrKeyIndex, offset) = data.asUint32CdUnchecked(offset);

          bytes memory shardData = _getSchnorrRawShardData(schnorrKeyIndex);

          result = abi.encodePacked(result, shardData);
        } else {
          revert InvalidOpcode(offset);
        }
      }

      require(offset == data.length, InvalidDataLength());

      return result;
    }
  }

  function _updateShardId(bytes calldata data, uint256 offset) internal returns (uint256 newOffset) {
    uint32 schnorrKeyIndex;
		uint32 expirationTime;
		bytes32 newSchnorrId;
		uint8 signerIndex;
		bytes32 r;
		bytes32 s;
		uint8 v;

		(schnorrKeyIndex, offset) = data.asUint32CdUnchecked(offset);
		(expirationTime, offset) = data.asUint32CdUnchecked(offset);
		(newSchnorrId, offset) = data.asBytes32CdUnchecked(offset);
		(signerIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);

		// We only allow registrations for the current threshold key
		require(schnorrKeyIndex + 1 == _getSchnorrKeyCount(), InvalidKeyIndex());

		// Verify the message is not expired
		require(expirationTime > block.timestamp, SignatureExpired());

    // Get the shard data range associated with the schnorr key
		(, uint8 shardCount, uint40 shardBase, uint32 multisigKeyIndex) = _getSchnorrExtraInfo(schnorrKeyIndex);
    require(signerIndex < shardCount, InvalidSignerIndex());

    // TODO: We could save a bit of gas by only codecopying the key we need
    // TODO: Should we check the expiration time or key count here too?
		(, uint256 keyDataOffset,) = _getMultisigKeyData(multisigKeyIndex);

    address expected;
    assembly ("memory-safe") {
      expected := mload(add(keyDataOffset, shl(5, signerIndex)))
    }

		// Verify the signature
		// We're not doing replay protection with the signature itself so we don't care about
		// verifying only canonical (low s) signatures.
		bytes32 digest = getRegisterGuardianDigest(schnorrKeyIndex, expirationTime, newSchnorrId);
		address signatory = ecrecover(digest, v, r, s);
		require(signatory == expected, VerificationFailed());

		// Store the shard ID
		_setSchnorrShardId(shardBase, signerIndex, newSchnorrId);

    return offset;
  }

  function _appendSchnorrKeys(bytes calldata data, uint256 offset) internal returns (uint256 newOffset) {
    unchecked {
      // Decode the VAA
      uint8 version;
      uint32 multisigKeyIndex;
      uint8 signatureCount;

      (version, offset) = data.asUint8CdUnchecked(offset);
      (multisigKeyIndex, offset) = data.asUint32CdUnchecked(offset);
      (signatureCount, offset) = data.asUint8CdUnchecked(offset);

      uint256 signatureOffset = offset;
      uint256 envelopeOffset = signatureOffset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;

      uint16 emitterChainId;
      bytes32 emitterAddress;

      (emitterChainId, offset) = data.asUint16CdUnchecked(envelopeOffset + VaaLib.ENVELOPE_EMITTER_CHAIN_ID_OFFSET);
      (emitterAddress, offset) = data.asBytes32CdUnchecked(offset);

      bytes32 module;
      uint8 action;
      uint32 newSchnorrKeyIndex;
      uint256 newSchnorrKey;
      uint32 expirationDelaySeconds;

      (module, offset) = data.asBytes32CdUnchecked(envelopeOffset + VaaLib.ENVELOPE_SIZE);
      (action, offset) = data.asUint8CdUnchecked(offset);

      (newSchnorrKeyIndex, offset) = data.asUint32CdUnchecked(offset);
      (newSchnorrKey, offset) = data.asUint256CdUnchecked(offset);
      (expirationDelaySeconds, offset) = data.asUint32CdUnchecked(offset);

      // Decode the pubkey
      uint256 px = newSchnorrKey >> 1;

      // Load current multisig key data
      uint256 currentMultisigKeyCount = _getMultisigKeyCount();
      uint256 currentMultisigKeyIndex = currentMultisigKeyCount - 1;
      (uint8 keyCount, uint256 keyDataOffset,) = _getMultisigKeyData(currentMultisigKeyIndex);

      require(version == 1, VaaLib.InvalidVersion(version));
      require(multisigKeyIndex == currentMultisigKeyIndex, InvalidMultisigKeyIndex());
      require(signatureCount == keyCount, InvalidSignatureCount());
      // NOTE: No need to check expiration, it's the current multisig key

      require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
      require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

      require(module == MODULE_VERIFICATION_V2, InvalidModule());
      require(action == ACTION_APPEND_SCHNORR_KEY, InvalidAction());
      require(newSchnorrKeyIndex == _getSchnorrKeyCount(), InvalidKeyIndex());
      require(eagerAnd(px != 0, px <= HALF_Q), InvalidKey());

      // Verify the signatures
      bytes32 vaaDoubleHash = data.calcVaaDoubleHashCd(envelopeOffset);
      _verifyMultisig(vaaDoubleHash, keyCount, keyDataOffset, signatureCount, data, signatureOffset);

      // If there is a previous schnorr key that is now expired, store the expiration time
      if (newSchnorrKeyIndex > 0) {
        uint32 newExpirationTime = uint32(block.timestamp) + expirationDelaySeconds;
        _setSchnorrExpirationTime(newSchnorrKeyIndex - 1, newExpirationTime);
      }

      // Store the new schnorr key data
      newOffset = _appendSchnorrKeyData(newSchnorrKey, multisigKeyIndex, signatureCount, data, offset);
    }
  }

  function _pullMultisigKeyData(uint32 limit) internal { // 298332
    unchecked {
      // Get the current state
      ICoreBridge coreBridge = _getCoreBridge();
			uint256 currentMultisigKeyIndex = coreBridge.getCurrentGuardianSetIndex();
			uint256 currentMultisigKeysLength = currentMultisigKeyIndex + 1;
			uint256 oldMultisigKeysLength = _getMultisigKeyCount();

      // If we've already pulled all the guardian sets, return
			if (currentMultisigKeysLength == oldMultisigKeysLength) return;

			// Check if we need to update the current guardian set
      if (oldMultisigKeysLength > 0) {
        // Pull and write the current guardian set expiration time
        uint32 updateIndex = uint32(oldMultisigKeysLength - 1);
        uint32 expirationTime = coreBridge.getGuardianSet(updateIndex).expirationTime;
        _setMultisigExpirationTime(updateIndex, expirationTime);
      }

			// Calculate the upper bound of the guardian sets to pull
      uint256 upper;
      assembly ("memory-safe") {
        let selector := and(iszero(iszero(limit)), lt(sub(currentMultisigKeysLength, oldMultisigKeysLength), limit))
        let selected := or(shl(32, currentMultisigKeysLength), add(oldMultisigKeysLength, limit))
        upper := and(shr(shl(5, selector), selected), 0xFFFFFFFF)
      }

      // Pull and append the guardian sets
      for (uint256 i = oldMultisigKeysLength; i < upper; i++) {
        // Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
        GuardianSet memory guardians = coreBridge.getGuardianSet(uint32(i));
        _appendMultisigKeyData(guardians.keys, guardians.expirationTime);
      }
		}
  }
}
