// SPDX-License-Identifier: MIT

pragma solidity ^0.8.28;

import {console} from "forge-std/console.sol";

import {ICoreBridge, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-solidity-sdk/libraries/VaaLib.sol";
import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";

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

uint8 constant UPDATE_SHARD_ID = 0;
uint8 constant UPDATE_APPEND_SCHNORR_KEY = 1;
uint8 constant UPDATE_PULL_MULTISIG_KEY_DATA = 2;

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
  uint256 constant internal HALF_Q = Q >> 1;

  error InvalidOpcode(uint256 offset);
  error DeploymentFailed();
  error InvalidPointer();
  error KeyDataExpired();

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
        mstore(0x00, 0x11052bb4) // InvalidPointer()
        revert(0x1c, 0x04)
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
        mstore(0x00, 0x30116425) // DeploymentFailed()
        revert(0x1c, 0x04)
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

  function _setSchnorrExpirationTime(uint32 index, uint32 expirationTime) internal {
    assembly ("memory-safe") {
      let ptr := add(SLOT_SCHNORR_KEY_EXTRA, index)
      sstore(ptr, or(and(sload(ptr), MASK_SCHNORR_EXTRA_EXPIRATION_TIME), expirationTime))
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
      let writePtr := add(SLOT_SCHNORR_SHARD_DATA, keyIndex)
      for {let i := 0} lt(i, shardCount) {i := add(i, 1)} {
        sstore(writePtr, calldataload(readPtr))
        sstore(add(writePtr, 1), calldataload(add(readPtr, 0x20)))
        writePtr := add(writePtr, 2)
        readPtr := add(readPtr, 0x40)
      }

      // Return the new offset
      newOffset := sub(readPtr, shardData.offset)
    }
  }

  function _verifyVaa(bytes calldata data) internal view returns (bool valid, uint256 envelopeOffset) {
    uint8 version;
    uint32 keyIndex;

    assembly ("memory-safe") {
      version := shr(248, calldataload(data.offset))
      keyIndex := shr(224, calldataload(add(data.offset, 1)))
    }

    if (version == 2) {
      assembly ("memory-safe") {
        // Decode the signature
        envelopeOffset := 57
        let r := shr(96, calldataload(add(data.offset, 5)))
        let s := calldataload(add(data.offset, 25))
        let validSignature := and(not(iszero(r)), lt(s, Q))

        // Compute the double hash of the VAA
        let buffer := mload(0x40)
        let envelopeLength := sub(data.length, envelopeOffset)
        calldatacopy(buffer, add(data.offset, envelopeOffset), envelopeLength)
        let singleHash := keccak256(buffer, envelopeLength)
        mstore(0, singleHash)
        let doubleHash := keccak256(0, 32)

        // Load the key and validate the expiration time
        let pubkey := sload(add(SLOT_SCHNORR_KEY_DATA, keyIndex))
        let expirationTime := and(sload(add(SLOT_SCHNORR_KEY_EXTRA, keyIndex)), 0xFFFFFFFF)
        let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))

        // Compute the challenge value
        let px := shr(1, pubkey)
        let validPubkey := not(iszero(px))
        let parity := and(pubkey, 1)
        mstore(buffer, px)
        mstore8(add(buffer, 32), parity)
        mstore(add(buffer, 33), doubleHash)
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
        valid := and(expirationTimeValid, and(validPubkey, and(validSignature, recoveredValid)))
      }
    } else if (version == 1) {
      bytes32 digest;
      uint8 signatureCount;

      assembly ("memory-safe") {
        // Decode the signature count
        signatureCount := shr(248, calldataload(add(data.offset, VAA_MULTISIG_SIGNATURE_COUNT_OFFSET)))
        envelopeOffset := add(VAA_MULTISIG_SIGNATURE_OFFSET, mul(signatureCount, VAA_MULTISIG_SIGNATURE_SIZE))

        // Compute the double hash of the VAA
        let buffer := mload(0x40)
        let envelopeLength := sub(data.length, envelopeOffset)
        calldatacopy(buffer, add(data.offset, envelopeOffset), envelopeLength)
        let singleHash := keccak256(buffer, envelopeLength)
        mstore(0, singleHash)
        digest := keccak256(0, 32)
      }

      // Load the key data and validate the expiration time
      // TODO: We should probably inline the rest of this, even though it requires copying the code, I don't trust the compiler
      (uint8 keyCount, uint256 keyDataOffset, uint32 expirationTime) = _getMultisigKeyData(keyIndex);
      require(expirationTime == 0 || expirationTime > block.timestamp, KeyDataExpired());

      // Verify the signatures
      valid = _verifyMultisig(digest, keyCount, keyDataOffset, signatureCount, data, VAA_MULTISIG_SIGNATURE_OFFSET);
    } else {
      valid = false;
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
}

contract Verification is VerificationCore {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error InvalidMultisigKeyIndex();
  error InvalidSignatureCount();
  error InvalidGovernanceChain();
  error InvalidGovernanceAddress();
  error InvalidModule();
  error InvalidAction();
  error InvalidKeyIndex();
  error InvalidKey();
  error VerificationFailed();

  constructor(
    ICoreBridge coreBridge,
    uint32 initialMultisigKeyCount,
    uint32 initialSchnorrKeyCount,
    uint32 initialMultisigKeyPullLimit
  ) VerificationCore(coreBridge, initialMultisigKeyCount, initialSchnorrKeyCount) {
    _pullMultisigKeyData(initialMultisigKeyPullLimit);
  }

  function verifyVaa_U7N5(bytes calldata data) external view {
    (bool valid,) = _verifyVaa(data);
    if (!valid) {
      revert VerificationFailed();
    }
  }

  function update(bytes calldata data) external {
    unchecked {
      uint256 offset = 0;

      while (offset < data.length) {
        uint8 opcode;

        (opcode, offset) = data.asUint8CdUnchecked(offset);

        if (opcode == UPDATE_SHARD_ID) {
          offset = _updateShardId(data, offset);
        } else if (opcode == UPDATE_APPEND_SCHNORR_KEY) {
          offset = _appendSchnorrKeys(data, offset);
        } else if (opcode == UPDATE_PULL_MULTISIG_KEY_DATA) {
          uint32 limit;
          (limit, offset) = data.asUint32CdUnchecked(offset);
          _pullMultisigKeyData(limit);
        } else {
          revert InvalidOpcode(offset);
        }
      }
    }
  }

  function _updateShardId(bytes calldata data, uint256 offset) internal returns (uint256 newOffset) {
    // TODO: Implement
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
      (uint8 keyCount, uint256 keyDataOffset, uint32 expirationTime) = _getMultisigKeyData(currentMultisigKeyIndex);

      require(version == 1, VaaLib.InvalidVersion(version));
      require(multisigKeyIndex == currentMultisigKeyIndex, InvalidMultisigKeyIndex());
      require(signatureCount == keyCount, InvalidSignatureCount());
      // NOTE: No need to check expiration, it's the current multisig key

      require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
      require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

      require(module == MODULE_VERIFICATION_V2, InvalidModule());
      require(action == ACTION_APPEND_SCHNORR_KEY, InvalidAction());
      require(newSchnorrKeyIndex == _getSchnorrKeyCount(), InvalidKeyIndex());
      require(px != 0 && px <= HALF_Q, InvalidKey());

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
