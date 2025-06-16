// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";

import {eagerAnd, eagerOr} from "wormhole-solidity-sdk/Utils.sol";
import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {ICoreBridge, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {CoreBridgeLib} from "wormhole-solidity-sdk/libraries/CoreBridge.sol";
import {UncheckedIndexing} from "wormhole-solidity-sdk/libraries/UncheckedIndexing.sol";
import {VaaLib, VaaBody} from "wormhole-solidity-sdk/libraries/VaaLib.sol";

import {EIP712Encoding} from "./EIP712Encoding.sol";
import {ExtStore} from "./ExtStore.sol";

enum VerificationError {
	NoError, // 0
	SignatureMismatch, // 1
	InvalidSignature, // 2

	InvalidVAAVersion, // 3
	KeyDataExpired
}

abstract contract VerificationCore {
	using BytesParsing for bytes;
	using VaaLib for bytes;
	using UncheckedIndexing for address[];

	error FailedVerification(VerificationError errorCode);

	// Curve order for secp256k1
  uint256 constant internal Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant internal HALF_Q = Q >> 1;

	function _decodePubkey(uint256 pubkey) internal pure returns (uint256 px, uint8 parity) {
		unchecked {
			parity = uint8((pubkey & 1) + VaaLib.SIGNATURE_RECOVERY_MAGIC);
			px = pubkey >> 1;
		}
	}

	function _verifySchnorr(
		bytes32 messageDigest,
		uint256 pubkey,
		bytes calldata signature,
		uint256 offset
	) internal view {
		unchecked {
			// Calculate the challenge value
			bool valid;
			bool validSignature;

			assembly ("memory-safe") {
				offset := add(signature.offset, offset)
				let r := shr(96, calldataload(offset))
				let s := calldataload(add(offset, 20))

				let ptr := mload(0x40)
				let px := shr(1, pubkey)
				let parity := and(pubkey, 1)
				mstore(ptr, px)
				mstore8(add(ptr, 32), parity)
				mstore(add(ptr, 33), messageDigest)
				mstore(add(ptr, 65), shl(96, r))
				let e := keccak256(ptr, 85)

        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
				mstore(ptr, sub(Q, mulmod(px, s, Q)))
				mstore(add(ptr, 32), add(parity, 27))
				mstore(add(ptr, 64), px)
				mstore(add(ptr, 96), mulmod(px, e, Q))
				let success := staticcall(gas(), 0x01, ptr, 128, ptr, 0x20)
				let recovered := mload(ptr)
				validSignature := and(not(iszero(r)), lt(s, Q))
				valid := and(validSignature, and(success, eq(r, recovered)))
			}
      
      require(valid, FailedVerification(validSignature ? VerificationError.SignatureMismatch : VerificationError.InvalidSignature));
		}
	}

	function _verifyMultisig(
		bytes32 messageDigest,
		address[] memory publicKeys,
		uint256 signatureCount,
		bytes calldata signatures,
		uint256 offset
	) internal pure {
		unchecked {
			// Verify the signatures;
			uint256 guardianCount = publicKeys.length - 1;
      uint256 usedSignerBitfield;
      
      for (uint256 i = 0; i < signatureCount; i++) {
        // Decode the guardian index, r, s, and v
        uint256 signerIndex;
				bytes32 r;
				bytes32 s;
				uint8 v;

        (signerIndex, r, s, v, offset) = signatures.decodeGuardianSignatureCdUnchecked(offset);

        // Verify the signature
        address signatory = ecrecover(messageDigest, v, r, s);
        address signer = publicKeys.readUnchecked(signerIndex);

        // Check that:
        // * no guardian indices are included twice
        // * no guardian indices are out of bounds
        // * the signatory is the guardian at the given index

				uint256 signerFlag = 1 << signerIndex;

        bool failed = eagerOr(
          eagerOr(
            (usedSignerBitfield & signerFlag) != 0,
            signerIndex > guardianCount
          ),
          signatory != signer
        );
        
        require(!failed, FailedVerification(VerificationError.SignatureMismatch));

        usedSignerBitfield |= signerFlag;
      }
		}
	}
}

abstract contract VerificationStateMultisig is ExtStore {
	using BytesParsing for bytes;

	ICoreBridge internal immutable _coreBridge;

	uint32[] private _multisigExpirationTimes;

	constructor(ICoreBridge coreBridge, uint32 initMultisigKeyIndex) {
		_coreBridge = coreBridge;

		require(initMultisigKeyIndex <= _coreBridge.getCurrentGuardianSetIndex());
		// All previous guardian sets will have expiration timestamp 0
		// FIXME: Is this okay? It seems like it would validate VAAs from before the initMultisigKeyIndex
		assembly ("memory-safe") {
			sstore(_multisigExpirationTimes.slot, initMultisigKeyIndex)
		}
	}

	function _getMultisigKeyCount() internal view returns (uint256) {
		return _multisigExpirationTimes.length;
	}

	function _getMultisigKeyData(uint32 multisigKeyIndex) internal view returns (uint32 expirationTime, address[] memory keys) {
		unchecked {
			expirationTime = _multisigExpirationTimes[multisigKeyIndex];
      bytes memory data = _extRead(multisigKeyIndex);

      // Convert the guardian set data to an array of addresses
      // NOTE: The `data` array is temporary and is invalid after this block
      assembly ("memory-safe") {
        keys := data
        mstore(keys, shr(5, mload(keys)))
      }
		}
	}

	function _appendMultisigKeyData(bytes memory data, uint32 expirationTime) internal {
		_multisigExpirationTimes.push(expirationTime);
		_extWrite(data);
	}

	function _setMultisigExpirationTime(uint32 multisigKeyIndex, uint32 expirationTime) internal {
		_multisigExpirationTimes[multisigKeyIndex] = expirationTime;
	}
}

struct ShardData {
	bytes32 shard;
	bytes32 id;
}

abstract contract VerificationStateSchnorr {
	using BytesParsing for bytes;

	struct SchnorrKeyData {
		uint256 pubkey;
		uint32 expirationTime;
		uint8 shardCount;
		uint40 shardBase;
		uint32 multisigKeyIndex;
	}

	error InvalidShardIndex();

	SchnorrKeyData[] private _schnorrKeyData;
	ShardData[] private _schnorrShardData;

	function _getSchnorrKeyCount() internal view returns (uint256) {
		return _schnorrKeyData.length;
	}

	function _getSchnorrKeyData(uint32 keyIndex) internal view returns (SchnorrKeyData memory) {
		return _schnorrKeyData[keyIndex];
	}

	function _appendSchnorrKeyData(uint256 pubkey, uint32 multisigKeyIndex, uint8 shardCount, bytes calldata shardData, uint256 offset) internal returns (uint256 newOffset) {
		// Append the key data
		_schnorrKeyData.push(SchnorrKeyData({
			pubkey: pubkey,
			expirationTime: 0,
			shardCount: shardCount,
			shardBase: uint40(_schnorrShardData.length),
			multisigKeyIndex: multisigKeyIndex
		}));

		// Append the shard data
		for (uint256 i = 0; i < shardCount; i++) {
			bytes32 shard;
			bytes32 id;

			(shard, offset) = shardData.asBytes32CdUnchecked(offset);
			(id, offset) = shardData.asBytes32CdUnchecked(offset);

			_schnorrShardData.push(ShardData({
				shard: shard,
				id: id
			}));
		}

		return offset;
	}

	function _setSchnorrExpirationTime(uint32 keyIndex, uint32 expirationTime) internal {
		_schnorrKeyData[keyIndex].expirationTime = expirationTime;
	}

	function _getSchnorrShardData(uint32 keyIndex) internal view returns (ShardData[] memory) {
		unchecked {
			SchnorrKeyData memory keyData = _getSchnorrKeyData(keyIndex);
			uint8 shardCount = keyData.shardCount;
			uint40 shardBase = keyData.shardBase;

			ShardData[] memory shardData = new ShardData[](shardCount);
			for (uint256 i = 0; i < shardCount; i++) {
				shardData[i] = _schnorrShardData[shardBase + i];
			}

			return shardData;
		}
	}

	function _setSchnorrShardId(SchnorrKeyData memory keyData, uint8 shardIndex, bytes32 id) internal {
		unchecked {
			require(shardIndex < keyData.shardCount, InvalidShardIndex());
			_schnorrShardData[keyData.shardBase + shardIndex].id = id;
		}
	}
}

abstract contract VerificationOptions {
	error InvalidInputType(uint8 inputType);
	error InvalidOutputType(uint8 outputType);

	uint8 constant private VERIFY_INPUT_MASK = 0x0F;
	uint8 constant private VERIFY_ERROR_MASK = 0xF0;

	uint256 constant internal VAA_COMMON_HEADER_LENGTH = 1 + 4;
	uint256 constant internal SCHNORR_SIGNATURE_LENGTH = 20 + 32;
	uint256 constant internal SCHNORR_VAA_HEADER_LENGTH = VAA_COMMON_HEADER_LENGTH + SCHNORR_SIGNATURE_LENGTH;

	function _decodeOptions(uint8 options) internal pure returns (uint8 inputType, uint8 outputType) {
		inputType = options & VERIFY_INPUT_MASK;
		outputType = options & VERIFY_ERROR_MASK;
	}
}

abstract contract VerificationSingle is VerificationOptions, VerificationStateMultisig, VerificationStateSchnorr, VerificationCore {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	function verifyVaaDecodeEssentials_gRd6(bytes calldata data) public view returns (uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) {
		uint256 envelopeOffset = _verifyVaa(data);
		return _decodeVaaEssentials(data, envelopeOffset);
	}

	function verifyVaa_U7N5(bytes calldata data) public view {
		_verifyVaa(data);
	}
	
	function verifyVaaDecodeBody(bytes calldata data) public view returns (VaaBody memory result) {
		uint256 envelopeOffset = _verifyVaa(data);
		uint payloadOffset;

    (
			result.envelope.timestamp,
			result.envelope.nonce,
			result.envelope.emitterChainId,
			result.envelope.emitterAddress,
			result.envelope.sequence,
			result.envelope.consistencyLevel,
			payloadOffset
		) = data.decodeVaaEnvelopeCdUnchecked(envelopeOffset);

    result.payload = data.decodeVaaPayloadCd(payloadOffset);
	}

	function verifyHashAndHeader(bytes32 digest, bytes calldata header) public view {
		unchecked {
			uint256 offset = 0;
			uint8 version;
			uint32 keyIndex;

			(version, offset) = header.asUint8CdUnchecked(offset);
			(keyIndex, offset) = header.asUint32CdUnchecked(offset);

			if (version == 2) {
				SchnorrKeyData memory keyData = _getSchnorrKeyData(keyIndex);
				uint32 expirationTime = keyData.expirationTime;
				require(eagerOr(expirationTime == 0, expirationTime > block.timestamp), FailedVerification(VerificationError.KeyDataExpired));

				_verifySchnorr(digest, keyData.pubkey, header, offset);
			} else if (version == 1) {
				uint8 signatureCount;

				(signatureCount, offset) = header.asUint8CdUnchecked(offset);
				
				(uint32 expirationTime, address[] memory keys) = _getMultisigKeyData(keyIndex);
				require(eagerOr(expirationTime == 0, expirationTime > block.timestamp), FailedVerification(VerificationError.KeyDataExpired));

				_verifyMultisig(digest, keys, signatureCount, header, offset);
			} else {
				revert VaaLib.InvalidVersion(version);
			}
		}
	}

	function _verifyVaa(bytes calldata data) public view returns (uint256 envelopeOffset) {
		unchecked {
			uint256 offset = 0;
			uint8 version;
			uint32 keyIndex;

			(version, offset) = data.asUint8CdUnchecked(offset);
			(keyIndex, offset) = data.asUint32CdUnchecked(offset);

			if (version == 2) {
				envelopeOffset = SCHNORR_VAA_HEADER_LENGTH;
				bytes32 digest = data.calcVaaDoubleHashCd(envelopeOffset);
				SchnorrKeyData memory keyData = _getSchnorrKeyData(keyIndex);
				uint32 expirationTime = keyData.expirationTime;
				require(eagerOr(expirationTime == 0, expirationTime > block.timestamp), FailedVerification(VerificationError.KeyDataExpired));

				_verifySchnorr(digest, keyData.pubkey, data, offset);
			} else if (version == 1) {
				uint8 signatureCount;

				(signatureCount, offset) = data.asUint8CdUnchecked(offset);

				envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
				bytes32 digest = data.calcVaaDoubleHashCd(envelopeOffset);

				(uint32 expirationTime, address[] memory keys) = _getMultisigKeyData(keyIndex);
				require(eagerOr(expirationTime == 0, expirationTime > block.timestamp), FailedVerification(VerificationError.KeyDataExpired));

				_verifyMultisig(digest, keys, signatureCount, data, offset);
			} else {
				revert VaaLib.InvalidVersion(version);
			}
		}
	}

	function _decodeVaaEssentials(bytes calldata data, uint256 envelopeOffset) internal pure returns (uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) {
		unchecked {
			// NOTE: We can't use the VaaLib version of this because it checks the version field as well
			uint256 offset = envelopeOffset + VaaLib.ENVELOPE_EMITTER_CHAIN_ID_OFFSET;
			(emitterChainId, offset) = data.asUint16CdUnchecked(offset);
			(emitterAddress, offset) = data.asBytes32CdUnchecked(offset);
			(sequence,             ) = data.asUint32CdUnchecked(offset);

			uint payloadOffset = envelopeOffset + VaaLib.ENVELOPE_SIZE;
			payload = data.decodeVaaPayloadCd(payloadOffset);
		}
	}
}

abstract contract VerificationCompressed is VerificationOptions, VerificationStateMultisig, VerificationStateSchnorr, VerificationCore {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	function verifyVaaCompressed() external view returns (bytes memory) {
		unchecked {
			uint256 offset = 4;
			uint8 options;

			(options, offset) = msg.data.asUint8CdUnchecked(offset);

		}
	}
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

contract Verification is VerificationSingle, EIP712Encoding {
	using BytesParsing for bytes;
	using VaaLib for bytes;
	using UncheckedIndexing for address[];
	using {BytesParsing.checkLength} for uint;

	error InvalidOpcode(uint8 opcode);

	error InvalidKeyIndex();
	error SignatureExpired();
	error InvalidGuardianIndex();
	error InvalidGovernanceChain();
	error InvalidGovernanceAddress();
	error InvalidModule();
	error InvalidAction();
	error InvalidKey();
	error QuorumNotMet();
	error InvalidAppendSchnorrKeyMessageLength();

	constructor(ICoreBridge coreBridge, uint32 initMultisigKeyIndex, uint32 pullLimit) VerificationStateMultisig(coreBridge, initMultisigKeyIndex) {
		_pullMultisigKeyData(pullLimit);
	}

	// FIXME: Use raw dispatcher even though it's not a loop? Or just add the loop?
	function update(bytes calldata data) public {
		(uint8 opcode,) = data.asUint8CdUnchecked(0);

		if (opcode == UPDATE_SHARD_ID) {
			_updateShardId(data, 1);
		} else if (opcode == UPDATE_APPEND_SCHNORR_KEY) {
			_appendSchnorrKeys(data, 1);
		} else if (opcode == UPDATE_PULL_MULTISIG_KEY_DATA) {
			uint32 pullLimit;
			(pullLimit,) = data.asUint32CdUnchecked(1);
			_pullMultisigKeyData(pullLimit);
		} else {
			revert InvalidOpcode(opcode);
		}
	}

	function _updateShardId(bytes calldata data, uint256 offset) private {
		uint32 schnorrKeyIndex;
		uint32 expirationTime;
		bytes32 guardianId;
		uint8 signerIndex;
		bytes32 r;
		bytes32 s;
		uint8 v;

		(schnorrKeyIndex, offset) = data.asUint32CdUnchecked(offset);
		(expirationTime, offset) = data.asUint32CdUnchecked(offset);
		(guardianId, offset) = data.asBytes32CdUnchecked(offset);
		(signerIndex, r, s, v, offset) = data.decodeGuardianSignatureCdUnchecked(offset);

		// We only allow registrations for the current threshold key
		require(schnorrKeyIndex == _getSchnorrKeyCount(), InvalidKeyIndex());

		// Verify the message is not expired
		require(expirationTime > block.timestamp, SignatureExpired());

		// Get the guardian set for the threshold key
		SchnorrKeyData memory keyData = _getSchnorrKeyData(schnorrKeyIndex);
		uint32 multisigKeyIndex = keyData.multisigKeyIndex;
		(, address[] memory keys) = _getMultisigKeyData(multisigKeyIndex); // TODO: We could save a bit of gas by only codecopying the key we need
		// TODO: Verify the guardian set is still valid? What about for the verification path?
		// We can't afford to check it there, so I'm skipping it here for now too

		// Verify the signature
		// We're not doing replay protection with the signature itself so we don't care about
		// verifying only canonical (low s) signatures.
		bytes32 digest = getRegisterGuardianDigest(schnorrKeyIndex, expirationTime, guardianId);
		address signatory = ecrecover(digest, v, r, s);
		require(signatory == keys.readUnchecked(signerIndex), FailedVerification(VerificationError.SignatureMismatch));

		// Store the shard ID
		_setSchnorrShardId(keyData, signerIndex, guardianId);
	}

	function _appendSchnorrKeys(bytes calldata data, uint256 offset) private {
		unchecked {
			while (offset < data.length) {
				uint16 encodedVaaLength;
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

				(encodedVaaLength, offset) = data.asUint16CdUnchecked(offset);
				uint256 baseOffset = offset;
				(version, offset) = data.asUint8CdUnchecked(offset);
				(multisigKeyIndex, offset) = data.asUint32CdUnchecked(offset);
				(signatureCount, offset) = data.asUint8CdUnchecked(offset);

				uint256 signaturesOffset = offset;
				uint256 envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;

				(
					,
					,
					emitterChainId,
					emitterAddress,
					,
					,
					offset
				) = data.decodeVaaEnvelopeCdUnchecked(envelopeOffset);

				(module, offset) = data.asBytes32MemUnchecked(offset);
				(action, offset) = data.asUint8MemUnchecked(offset);

				(newSchnorrKeyIndex, offset) = data.asUint32MemUnchecked(offset);
				(newSchnorrKey, offset) = data.asUint256MemUnchecked(offset);
				(expirationDelaySeconds, offset) = data.asUint32MemUnchecked(offset);

				// Get the current guardian set index
				uint32 currentMultisigKeyIndex = _coreBridge.getCurrentGuardianSetIndex();
				(, address[] memory guardians) = _getMultisigKeyData(currentMultisigKeyIndex);

				// Decode the pubkey
				(uint256 px,) = _decodePubkey(newSchnorrKey);

				// Verify the VAA
				require(version == 1, VaaLib.InvalidVersion(version));
				require(multisigKeyIndex == currentMultisigKeyIndex, InvalidKeyIndex());
				require(signatureCount == guardians.length, QuorumNotMet());

				require(emitterChainId == CHAIN_ID_SOLANA, InvalidGovernanceChain());
				require(emitterAddress == GOVERNANCE_ADDRESS, InvalidGovernanceAddress());

				require(module == MODULE_VERIFICATION_V2, InvalidModule());
				require(action == ACTION_APPEND_SCHNORR_KEY, InvalidAction());
				require(newSchnorrKeyIndex == _getSchnorrKeyCount(), InvalidKeyIndex());
				require(eagerAnd(px != 0, px <= HALF_Q), InvalidKey());

				// Verify the signatures
				bytes32 vaaDoubleHash = data.calcVaaDoubleHashCd(envelopeOffset);
				_verifyMultisig(vaaDoubleHash, guardians, signatureCount, data, signaturesOffset);

				// If there is a previous schnorr key that is now expired, store the expiration time
				if (newSchnorrKeyIndex > 0) {
					uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
					_setSchnorrExpirationTime(newSchnorrKeyIndex - 1, expirationTime);
				}

				// Store the new schnorr key data
				_appendSchnorrKeyData(newSchnorrKey, multisigKeyIndex, signatureCount, data, offset);

				// Update the offset to the next encoded VAA
				offset = baseOffset + encodedVaaLength;
			}
		}
	}

	function _pullMultisigKeyData(uint32 limit) private {
		unchecked {
			uint256 currentMultisigKeyIndex = _coreBridge.getCurrentGuardianSetIndex();
			uint256 currentMultisigKeysLength = currentMultisigKeyIndex + 1;
			uint256 oldMultisigKeysLength = _getMultisigKeyCount();

			if (currentMultisigKeysLength == oldMultisigKeysLength) return;

			// Check if we need to update the current guardian set
      if (oldMultisigKeysLength > 0) {
        // Pull and write the current guardian set expiration time
        uint32 updateIndex = uint32(oldMultisigKeysLength - 1);
        (, uint32 expirationTime) = _pullMultisigKeyDataEntry(updateIndex);
        _setMultisigExpirationTime(updateIndex, expirationTime);
      }

			// Calculate the upper bound of the guardian sets to pull
      uint256 upper = eagerOr(limit == 0, currentMultisigKeysLength - oldMultisigKeysLength < limit)
        ? currentMultisigKeysLength : oldMultisigKeysLength + limit;

      // Pull and append the guardian sets
      for (uint256 i = oldMultisigKeysLength; i < upper; i++) {
        // Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
        (bytes memory data, uint32 expirationTime) = _pullMultisigKeyDataEntry(uint32(i));
        _appendMultisigKeyData(data, expirationTime);
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
