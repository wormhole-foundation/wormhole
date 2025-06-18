// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";

import {ICoreBridge, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {VaaLib, VaaBody} from "wormhole-solidity-sdk/libraries/VaaLib.sol";

// Governance emitter address
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_SCHNORR_KEY = 0x01;

// Update opcodes
uint256 constant UPDATE_SHARD_ID = 0;
uint256 constant UPDATE_APPEND_SCHNORR_KEY = 1;
uint256 constant UPDATE_PULL_MULTISIG_KEY_DATA = 2;

contract AssemblyOptimized {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  error VerificationFailure();
  error InvalidOpcode(uint8 opcode);

  error InvalidKeyIndex();
  error QuorumNotMet();
  error InvalidGovernanceChain();
  error InvalidGovernanceAddress();
  error InvalidModule();
  error InvalidAction();
  error InvalidSchnorrKey();

  // TODO: Encode range instead of length for schnorr and multisig, so we can set an initial offset
  uint256 constant internal _schnorrKeyCountOffset = 1000;
  uint256 constant internal _schnorrShardCountOffset = 1001;
  uint256 constant internal _multisigKeyCountOffset = 1002;

  uint256 constant internal _schnorrPubkeyOffset = 1 << 50;
  uint256 constant internal _schnorrDataOffset = 2 << 50;
  uint256 constant internal _schnorrShardDataOffset = 3 << 50;
  uint256 constant internal _multisigExpirationTimeOffset = 4 << 50;

  // Curve order for secp256k1
  uint256 constant internal Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant internal HALF_Q = Q >> 1;

  // Various offsets for decoding VAAs
  // NOTE: Can't use the VaaLib version of these because it makes the assembly blocks below unhappy
  uint256 constant internal VAA_MULTISIG_SIGNATURE_COUNT_OFFSET = 5;
  uint256 constant internal VAA_MULTISIG_SIGNATURE_OFFSET = 6;
  uint256 constant internal VAA_MULTISIG_SIGNATURE_R_OFFSET = 1;
  uint256 constant internal VAA_MULTISIG_SIGNATURE_S_OFFSET = 33;
  uint256 constant internal VAA_MULTISIG_SIGNATURE_V_OFFSET = 65;
  uint256 constant internal VAA_MULTISIG_SIGNATURE_LENGTH = 66;

  uint256 constant internal VAA_ENVELOPE_EMITTER_CHAIN_ID_OFFSET = 8;
  uint256 constant internal VAA_ENVELOPE_EMITTER_ADDRESS_OFFSET = 10;
  uint256 constant internal VAA_ENVELOPE_SEQUENCE_OFFSET = 42;
  uint256 constant internal VAA_ENVELOPE_PAYLOAD_OFFSET = 51;

  uint256 constant internal ABI_ARRAY_DATA_OFFSET = 32;

  ICoreBridge private immutable _coreBridge;

  constructor(
    ICoreBridge coreBridge,
    uint32 multisigKeyPullLimit
  ) {
    _coreBridge = coreBridge;

    _pullMultisigKeyData(multisigKeyPullLimit);
  }
  
  // TODO: Raw dispatcher _exec interface instead?
  function update(bytes calldata data) external {
    unchecked {
      uint256 offset = 0;
      while (offset < data.length) {
        uint8 opcode;

        (opcode, offset) = data.asUint8CdUnchecked(offset);

        if (opcode == 0) {
          offset = _updateSchnorrShardId(data, offset);
        } else if (opcode == 1) {
          offset = _appendSchnorrKeys(data, offset);
        } else if (opcode == 2) {
          uint32 pullLimit;
          (pullLimit, offset) = data.asUint32CdUnchecked(offset);
          _pullMultisigKeyData(pullLimit);
        } else {
          revert InvalidOpcode(opcode);
        }
      }
    }
  }

  function verifyVaa_U7N5(bytes calldata data) external view {
    _verifyVaa(data);
	}

  function verifyVaaDecodeEssentials_gRd6(bytes calldata data) external view returns (uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) {
    uint256 envelopeOffset = _verifyVaa(data);

    assembly ("memory-safe") {
      emitterChainId := shr(240, calldataload(add(envelopeOffset, VAA_ENVELOPE_EMITTER_CHAIN_ID_OFFSET)))
      emitterAddress := calldataload(add(envelopeOffset, VAA_ENVELOPE_EMITTER_ADDRESS_OFFSET))
      sequence := shr(192, calldataload(add(envelopeOffset, VAA_ENVELOPE_SEQUENCE_OFFSET)))

      let payloadOffset := add(envelopeOffset, VAA_ENVELOPE_PAYLOAD_OFFSET)
      let payloadLength := sub(data.length, payloadOffset)
      payload := mload(0x40)
      let payloadDataOffset := add(payload, ABI_ARRAY_DATA_OFFSET)
      mstore(0x40, add(payloadDataOffset, payloadLength))
      mstore(payload, payloadLength)
      calldatacopy(payloadDataOffset, payloadOffset, payloadLength)
    }
  }

  function verifyVaaDecodeBody(bytes calldata data) external view returns (VaaBody memory result) {
    uint256 envelopeOffset = _verifyVaa(data);
    result = data.decodeVaaBodyStructCd(envelopeOffset);
  }

  function _verifyVaa(bytes calldata data) internal view returns (uint256 envelopeOffset) {
    bool valid;

    assembly ("memory-safe") {
      let version := shr(248, calldataload(data.offset))
      let keyIndex := shr(224, calldataload(add(data.offset, 1)))

      switch version
        case 2 {
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
					let pubkey := sload(add(_schnorrPubkeyOffset, keyIndex))
					let expirationTime := and(sload(add(_schnorrDataOffset, keyIndex)), 0xFFFFFFFF)
					let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))

					// Compute the challenge value
					let px := shr(1, pubkey)
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
					valid := and(expirationTimeValid, and(validSignature, recoveredValid))
        }
        case 1 {          
          // Decode the signature count
					let signatureCount := shr(248, calldataload(add(data.offset, VAA_MULTISIG_SIGNATURE_COUNT_OFFSET)))
					envelopeOffset := add(VAA_MULTISIG_SIGNATURE_OFFSET, mul(signatureCount, VAA_MULTISIG_SIGNATURE_LENGTH))

					// Compute the double hash of the VAA
					let buffer := mload(0x40)
					let envelopeLength := sub(data.length, envelopeOffset)
					calldatacopy(buffer, add(data.offset, envelopeOffset), envelopeLength)
					let singleHash := keccak256(buffer, envelopeLength)
					mstore(0, singleHash)
					let doubleHash := keccak256(0, 32)

          // Validate the key index and expiration time
          let keyCount := sload(_multisigKeyCountOffset)
          let keyIndexValid := lt(keyIndex, keyCount)
          let expirationTime := sload(add(_multisigExpirationTimeOffset, keyIndex))
          let expirationTimeValid := or(iszero(expirationTime), gt(expirationTime, timestamp()))

          // Compute the key data address
          // TODO: Load this from an array instead of calculating it to save gas
          // TODO: No need to bounds check that array, since it'll be zero initialized and we check for empty blobs already
          let nonce := add(keyIndex, 1)
          let keyDataAddress
          switch lt(nonce, 0x80)
            case 1 {
              mstore(0x00, address())
              mstore8(0x0b, 0x94)
              mstore8(0x0a, 0xd6)
              mstore8(0x20, or(shl(7, iszero(nonce)), nonce))
              keyDataAddress := keccak256(0x0a, 0x17)
            }
            case 0 {
              // Get number of bytes in nonce
              // NOTE: We can cap this at 4 bytes because we limit the VAA key index to 4 bytes
              let i := add(add(add(1, gt(nonce, 0xFF)), gt(nonce, 0xFFFF)), gt(nonce, 0xFFFFFF))
              
              // Store in descending slot sequence to overlap the values correctly.
              mstore(i, nonce)
              mstore(0x00, shl(8, address()))
              mstore8(0x1f, add(0x80, i))
              mstore8(0x0a, 0x94)
              mstore8(0x09, add(0xd6, i))
              keyDataAddress := keccak256(0x09, add(0x17, i))
            }

          // Load the key data
          let keyDataCodeSize := extcodesize(keyDataAddress)
          let keyDataValid := gt(keyDataCodeSize, 0)

          let keyDataSize := sub(keyDataCodeSize, 1)
          let multisigSize := shr(5, keyDataSize)

          // Allocate enough 32 bytes words to cover the key data, and update the buffer to point to free memory
          // Then store the key data to the allocated memory
          let keyData := buffer
          buffer := add(buffer, and(add(keyDataSize, 0x3F), 0xFFE0))
          extcodecopy(keyDataAddress, keyData, 1, keyDataSize)

					// Decode the signatures and verify them
					let usedSignerBitfield := 0
          valid := and(and(keyIndexValid, expirationTimeValid), keyDataValid)
					
					let ptr := add(data.offset, envelopeOffset)
					for {let i := 0} lt(i, signatureCount) {i := add(i, 1)} {
						// Decode the next signature
						let signerIndex := shr(248, calldataload(ptr))
						let r := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_R_OFFSET))
						let s := calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_S_OFFSET))
						let v := shr(248, calldataload(add(ptr, VAA_MULTISIG_SIGNATURE_V_OFFSET)))

						// Call ecrecover
						mstore(buffer, doubleHash)
						mstore(add(buffer, 32), v)
						mstore(add(buffer, 64), r)
						mstore(add(buffer, 96), s)
						let success := staticcall(gas(), 0x01, buffer, 128, buffer, 0x20)

						// Validate the result
						let recovered := mload(buffer)
						let expected := mload(add(keyData, shl(5, signerIndex)))
						let signatureValid := eq(expected, recovered)
						let indexValid := lt(signerIndex, multisigSize)
						let signerFlag := shl(signerIndex, 1)
						let signerUsedValid := iszero(and(usedSignerBitfield, signerFlag))

						valid := and(valid, and(and(indexValid, signatureValid), signerUsedValid))
						usedSignerBitfield := or(usedSignerBitfield, signerFlag)
						ptr := add(ptr, 66)
					}
        }
        default {
          valid := 0
        }
    }

    require(valid, VerificationFailure());
  }

  function _updateSchnorrShardId(bytes calldata data, uint256 offset) internal returns (uint256 newOffset) {
    // TODO: Implement
    return offset;
  }

  function _appendSchnorrKeys(bytes calldata data, uint256 offset) internal returns (uint256 newOffset) {
    uint256 envelopeOffset = _verifyVaa(data[offset:]);

    // Decode the VAA for relevant fields(extra work, but it's not a critical path)
    uint8 version;
    uint32 multisigKeyIndex;
    uint8 signatureCount;

    uint16 emitterChainId;
    bytes32 emitterAddress;
    bytes32 module;
    uint8 action;
    uint32 newSchnorrKeyIndex;
    uint256 newSchnorrKey;
    uint32 expirationDelaySeconds;

    (version, offset) = data.asUint8CdUnchecked(offset);
    (multisigKeyIndex, offset) = data.asUint32CdUnchecked(offset);
    (signatureCount, offset) = data.asUint8CdUnchecked(offset);

    (emitterChainId, offset) = data.asUint16CdUnchecked(envelopeOffset + VAA_ENVELOPE_EMITTER_CHAIN_ID_OFFSET);
    (emitterAddress, offset) = data.asBytes32CdUnchecked(offset);

    (module, offset) = data.asBytes32CdUnchecked(envelopeOffset + VAA_ENVELOPE_PAYLOAD_OFFSET);
    (action, offset) = data.asUint8CdUnchecked(offset);
    (newSchnorrKeyIndex, offset) = data.asUint32CdUnchecked(offset);
    (newSchnorrKey, offset) = data.asUint256CdUnchecked(offset);
    (expirationDelaySeconds, offset) = data.asUint32CdUnchecked(offset);

    // Load extra info needed for the verification
    uint32 currentMultisigKeyCount = _getMultisigKeyCount();
    uint32 currentSchnorrKeyCount = _getSchnorrKeyCount();

    require(currentMultisigKeyCount > 0, InvalidKeyIndex());
    uint32 currentMultisigKeyIndex = currentMultisigKeyCount - 1;

    address keyDataAddress = _getMultisigDataAddress(multisigKeyIndex);
    uint8 currentMultisigSize;
    assembly ("memory-safe") {
      let codeSize := extcodesize(keyDataAddress)
      let keyDataSize := sub(codeSize, 1)
      currentMultisigSize := shr(5, keyDataSize)
    }

    uint256 px = newSchnorrKey >> 1;

    // Validate the message
    require(version == 1, VaaLib.InvalidVersion(version));
    require(multisigKeyIndex == currentMultisigKeyIndex, InvalidKeyIndex());
    require(signatureCount == currentMultisigSize, QuorumNotMet());

    require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
    require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

    require(module == MODULE_VERIFICATION_V2, InvalidModule());
    require(action == ACTION_APPEND_SCHNORR_KEY, InvalidAction());
    require(newSchnorrKeyIndex == currentSchnorrKeyCount, InvalidKeyIndex());
    require(px != 0 && px <= HALF_Q, InvalidSchnorrKey());

    // If there is a previous schnorr key that is now expired, store the expiration time
    if (newSchnorrKeyIndex > 0) {
      assembly ("memory-safe") {
        let expirationTime := add(timestamp(), expirationDelaySeconds)
        sstore(add(_schnorrDataOffset, sub(newSchnorrKeyIndex, 1)), expirationTime)
      }
    }

    // Store the new schnorr key data
    // Append the key data

    assembly ("memory-safe") {
      let currentShardCount := sload(_schnorrShardCountOffset)
      let newSchnorrKeyData := or(shl(32, currentMultisigSize), or(shl(40, currentShardCount), shl(80, currentMultisigKeyIndex)))
      sstore(add(_schnorrPubkeyOffset, newSchnorrKeyIndex), newSchnorrKey)
      sstore(add(_schnorrDataOffset, newSchnorrKeyIndex), newSchnorrKeyData)
      sstore(_schnorrKeyCountOffset, add(currentSchnorrKeyCount, 1))

      let ptr := add(data.offset, offset)
      let shardPtr := add(_schnorrShardDataOffset, shl(1, currentShardCount))
      for {let i := 0} lt(i, currentMultisigSize) {i := add(i, 1)} {
        sstore(shardPtr, calldataload(ptr))
        sstore(add(shardPtr, 1), calldataload(add(ptr, 32)))
        shardPtr := add(shardPtr, 2)
        ptr := add(ptr, 64)
      }

      sstore(_schnorrShardCountOffset, add(currentShardCount, currentMultisigSize))

      newOffset := sub(ptr, data.offset)
    }
  }

  function _getMultisigKeyCount() internal view returns (uint32 result) {
    assembly ("memory-safe") {
      result := sload(_multisigKeyCountOffset)
    }
  }

  function _getSchnorrKeyCount() internal view returns (uint32 result) {
    assembly ("memory-safe") {
      result := sload(_schnorrKeyCountOffset)
    }
  }

  function _getMultisigDataAddress(uint32 index) internal view returns (address result) {
    assembly ("memory-safe") {
      // Compute the key data address
      let nonce := add(index, 1)
      switch lt(nonce, 0x80)
        case 1 {
          mstore(0x00, address())
          mstore8(0x0b, 0x94)
          mstore8(0x0a, 0xd6)
          mstore8(0x20, or(shl(7, iszero(nonce)), nonce))
          result := keccak256(0x0a, 0x17)
        }
        case 0 {
          // Get number of bytes in nonce
          // NOTE: We can cap this at 4 bytes because we limit the VAA key index to 4 bytes
          let i := add(add(add(1, gt(nonce, 0xFF)), gt(nonce, 0xFFFF)), gt(nonce, 0xFFFFFF))
          
          // Store in descending slot sequence to overlap the values correctly.
          mstore(i, nonce)
          mstore(0x00, shl(8, address()))
          mstore8(0x1f, add(0x80, i))
          mstore8(0x0a, 0x94)
          mstore8(0x09, add(0xd6, i))
          result := keccak256(0x09, add(0x17, i))
        }
    }
  }

  function _pullMultisigKeyData(uint32 pullLimit) internal {
    unchecked {
      uint256 currentMultisigKeyIndex = _coreBridge.getCurrentGuardianSetIndex();
      uint256 currentMultisigKeyCount = currentMultisigKeyIndex + 1;
      uint256 oldMultisigKeyCount = _getMultisigKeyCount();

      if (currentMultisigKeyCount == oldMultisigKeyCount) return;

      if (oldMultisigKeyCount > 0) {
        uint32 updateIndex = uint32(oldMultisigKeyCount - 1);
        (, uint32 expirationTime) = _pullMultisigKeyDataEntry(updateIndex);
        
        assembly ("memory-safe") {
          sstore(add(_multisigExpirationTimeOffset, updateIndex), expirationTime)
        }
      }

      bool upperCond = pullLimit == 0 || (currentMultisigKeyCount - oldMultisigKeyCount < pullLimit);
      uint256 upper = upperCond ? currentMultisigKeyCount : oldMultisigKeyCount + pullLimit;

      for (uint256 i = oldMultisigKeyCount; i < upper; i++) {
        (bytes memory data, uint32 expirationTime) = _pullMultisigKeyDataEntry(uint32(i));

        assembly ("memory-safe") {
          // Store the expiration time
          sstore(add(_multisigExpirationTimeOffset, i), expirationTime)
          
          // Store the key data
          let originalDataLength := mload(data)
          let dataLength := add(data, 1)
          mstore(add(data, gt(dataLength, 0xFFFF)), or(0xfd61000080600a3d393df300, shl(0x40, dataLength)))
          let pointer := create(0, add(data, 0x15), add(dataLength, 0xA))
          if iszero(pointer) {
            mstore(0x00, 0x30116425)
            revert(0x1c, 0x04)
          }

          mstore(data, originalDataLength)
        }
      }

      assembly ("memory-safe") {
        sstore(_multisigKeyCountOffset, upper)
      }
    }
  }

  function _pullMultisigKeyDataEntry(uint32 index) private view returns (
    bytes memory data,
    uint32 expirationTime
  ) {
    // Get the guardian set from the core bridge
    // NOTE: The expiration time is copied from the core bridge,
    //       so any invalid guardian set will already be invalidated
    GuardianSet memory guardians = _coreBridge.getGuardianSet(index);
    expirationTime = guardians.expirationTime;

    // Convert the guardian set to a byte array
    // Result is stored in `data`
    // NOTE: The `keys` array is temporary and is invalid after this block
    address[] memory keys = guardians.keys;
    assembly ("memory-safe") {
      data := keys
      mstore(data, shl(5, mload(data)))
    }
  }
}
