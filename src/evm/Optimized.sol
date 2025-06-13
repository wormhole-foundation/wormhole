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
	) internal pure returns (VerificationError errorCode) {
		unchecked {
			// Decode the signature
			address r;
			uint256 s;

			(r, offset) = signature.asAddressCdUnchecked(offset);
      (s, offset) = signature.asUint256CdUnchecked(offset);

			// Decode the pubkey
			(uint256 px, uint8 parity) = _decodePubkey(pubkey);

			// Calculate the challenge value
      uint256 e = uint256(keccak256(abi.encodePacked(px, parity == 28, messageDigest, r)));

			// Calculate the recovered address
      address recovered = ecrecover(
        // NOTE: This is non-zero because for all k = px * s, Q > k % Q
        //       Therefore, Q - k % Q is always positive
        bytes32(Q - mulmod(px, s, Q)),
        parity,
        // NOTE: This is range checked in _decodeThresholdKeyUpdatePayload
        bytes32(px),
        bytes32(mulmod(px, e, Q))
      );

      // Verify that none of the preconditions were violated
      // NOTE: s < Q prevents signature malleability
      // NOTE: Non-zero r prevents confusion with ecrecover failure
      // NOTE: Non-zero check on s not needed, see the first argument of ecrecover
      bool validSignature = eagerAnd(r != address(0), s < Q);
      bool validRecovered = r == recovered;
      
      if (eagerAnd(validSignature, validRecovered)) {
				return VerificationError.NoError;
			}

			return validSignature ? VerificationError.SignatureMismatch : VerificationError.InvalidSignature;
		}
	}

	function _verifyMultisig(
		bytes32 messageDigest,
		address[] memory publicKeys,
		uint256 signatureCount,
		bytes calldata signatures,
		uint256 offset
	) internal pure returns (VerificationError errorCode) {
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
        
        if (failed) return VerificationError.SignatureMismatch;

        usedSignerBitfield |= signerFlag;
      }

			return VerificationError.NoError;
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

uint8 constant VERIFY_VAA_ESSENTIALS = 0;
uint8 constant VERIFY_VAA = 1;
uint8 constant VERIFY_VAA_BODY = 2;
uint8 constant VERIFY_HASH_AND_HEADER = 3;
uint8 constant VERIFY_HASH_AND_SCHNORR_SIGNATURE = 4;
uint8 constant VERIFY_HASH_AND_MULTISIG_SIGNATURE = 5;

uint8 constant VERIFY_ERROR_REVERT = 0 << 4;
uint8 constant VERIFY_ERROR_BOOL = 1 << 4;
uint8 constant VERIFY_ERROR_CODE = 2 << 4;

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

	function verify(uint8 options, bytes calldata data) public view returns (bytes memory) {
		(uint8 inputType, uint8 outputType) = _decodeOptions(options);

		// TODO: Binary tree?
		if (inputType < VERIFY_HASH_AND_HEADER) {
			return _verifyVaa(inputType, outputType, data);
		} else if (inputType == VERIFY_HASH_AND_HEADER) {
			return _verifyHashAndHeader(inputType, outputType, data);
		} else if (inputType == VERIFY_HASH_AND_SCHNORR_SIGNATURE) {
			return _verifyHashAndSchnorr(inputType, outputType, data);
		} else if (inputType == VERIFY_HASH_AND_MULTISIG_SIGNATURE) {
			return _verifyHashAndMultisig(inputType, outputType, data);
		} else {
			revert InvalidInputType(inputType);
		}
	}

	function _verifyVaa(uint8 inputType, uint8 outputType, bytes calldata data) internal view returns (bytes memory) {
		uint256 offset = 0;
		uint8 version;
		uint32 keyIndex;

		(version, offset) = data.asUint8CdUnchecked(offset);
		(keyIndex, offset) = data.asUint32CdUnchecked(offset);

		if (version == 2) {
			bytes32 vaaHash = data.calcVaaDoubleHashCd(SCHNORR_VAA_HEADER_LENGTH);
			return _verifyHashAndHeaderSchnorr(inputType, outputType, vaaHash, keyIndex, data, offset);
		} else if (version == 1) {
			uint8 signatureCount;

			(signatureCount, offset) = data.asUint8CdUnchecked(offset);

			uint256 envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
			bytes32 vaaHash = data.calcVaaDoubleHashCd(envelopeOffset);
			return _verifyHashAndHeaderMultisig(inputType, outputType, vaaHash, keyIndex, data, offset);
		} else {
			return _generateOutputABI(inputType, outputType, VerificationError.InvalidVAAVersion, data, offset);
		}
	}

	function _verifyHashAndHeader(uint8 inputType, uint8 outputType, bytes calldata data) internal view returns (bytes memory) {
		uint256 offset = 0;
		bytes32 digest;
		uint8 version;
		uint32 keyIndex;

		(digest, offset) = data.asBytes32CdUnchecked(offset);
		(version, offset) = data.asUint8CdUnchecked(offset);
		(keyIndex, offset) = data.asUint32CdUnchecked(offset);

		if (version == 2) {
			return _verifyHashAndHeaderSchnorr(inputType, outputType, digest, keyIndex, data, offset);
		} else if (version == 1) {
			return _verifyHashAndHeaderMultisig(inputType, outputType, digest, keyIndex, data, offset);
		} else {
			return _generateOutputABI(inputType, outputType, VerificationError.InvalidVAAVersion, data, offset);
		}
	}

	function _verifyHashAndHeaderSchnorr(uint8 inputType, uint8 outputType, bytes32 digest, uint32 keyIndex, bytes calldata data, uint256 offset) internal view returns (bytes memory) {
		SchnorrKeyData memory keyData = _getSchnorrKeyData(keyIndex);
		uint32 expirationTime = keyData.expirationTime;
		if (eagerAnd(expirationTime != 0, expirationTime < block.timestamp)) {
			return _generateOutputABI(inputType, outputType, VerificationError.KeyDataExpired, data, offset);
		}

		VerificationError errorCode = _verifySchnorr(digest, keyData.pubkey, data, offset);
		return _generateOutputABI(inputType, outputType, errorCode, data, offset);
	}

	function _verifyHashAndHeaderMultisig(uint8 inputType, uint8 outputType, bytes32 digest, uint32 keyIndex, bytes calldata data, uint256 offset) internal view returns (bytes memory) {
		uint8 signatureCount;

		(signatureCount, offset) = data.asUint8CdUnchecked(offset);

		(uint32 expirationTime, address[] memory keys) = _getMultisigKeyData(keyIndex);
		if (eagerAnd(expirationTime != 0, expirationTime < block.timestamp)) {
			return _generateOutputABI(inputType, outputType, VerificationError.KeyDataExpired, data, offset);
		}

		VerificationError errorCode = _verifyMultisig(digest, keys, signatureCount, data, offset);
		return _generateOutputABI(inputType, outputType, errorCode, data, offset);
	}

	function _verifyHashAndSchnorr(uint8 inputType, uint8 outputType, bytes calldata data) internal pure returns (bytes memory) {
		uint256 offset = 0;
		bytes32 vaaHash;
		uint256 pubkey;

		(vaaHash, offset) = data.asBytes32CdUnchecked(offset);
		(pubkey, offset) = data.asUint256CdUnchecked(offset);

		VerificationError errorCode = _verifySchnorr(vaaHash, pubkey, data, offset);
		return _generateOutputABI(inputType, outputType, errorCode, data, offset);
	}

	function _verifyHashAndMultisig(uint8 inputType, uint8 outputType, bytes calldata data) internal pure returns (bytes memory) {
		uint256 offset = 0;
		bytes32 vaaHash;
		uint8 guardianCount;
		address[] memory keys;
		uint8 signatureCount;

		(vaaHash, offset) = data.asBytes32CdUnchecked(offset);
		(guardianCount, offset) = data.asUint8CdUnchecked(offset);
		(signatureCount, offset) = data.asUint8CdUnchecked(offset);

		keys = new address[](guardianCount);
		for (uint256 i = 0; i < guardianCount; i++) {
			(keys[i], offset) = data.asAddressCdUnchecked(offset);
		}

		VerificationError errorCode = _verifyMultisig(vaaHash, keys, signatureCount, data, offset);
		return _generateOutputABI(inputType, outputType, errorCode, data, offset);
	}

	function _generateOutputABI(uint8 inputType, uint8 outputType, VerificationError errorCode, bytes calldata data, uint256 envelopeOffset) internal pure returns (bytes memory) {
		if (outputType == VERIFY_ERROR_REVERT) {
			if (errorCode != VerificationError.NoError) revert FailedVerification(errorCode);

			if (inputType == VERIFY_VAA_ESSENTIALS) {
				(uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) = _decodeVaaEssentials(data, envelopeOffset);
				return abi.encode(emitterChainId, emitterAddress, sequence, payload);
			} else if (inputType == VERIFY_VAA_BODY) {
				return abi.encode(data.decodeVaaBodyStructCd(envelopeOffset));
			} else {
				return new bytes(0);
			}
		} else if (outputType == VERIFY_ERROR_BOOL) {
			bool errorBool = errorCode == VerificationError.NoError;

			if (inputType == VERIFY_VAA_ESSENTIALS) {
				(uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) = _decodeVaaEssentials(data, envelopeOffset);
				return abi.encode(errorBool, emitterChainId, emitterAddress, sequence, payload);
			} else if (inputType == VERIFY_VAA_BODY) {
				return abi.encode(errorBool, data.decodeVaaBodyStructCd(envelopeOffset));
			} else {
				return abi.encode(errorBool);
			}
		} else if (outputType == VERIFY_ERROR_CODE) {
			if (inputType == VERIFY_VAA_ESSENTIALS) {
				(uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) = _decodeVaaEssentials(data, envelopeOffset);
				return abi.encode(errorCode, emitterChainId, emitterAddress, sequence, payload);
			} else if (inputType == VERIFY_VAA_BODY) {
				return abi.encode(errorCode, data.decodeVaaBodyStructCd(envelopeOffset));
			} else {
				return abi.encode(errorCode);
			}
		} else {
			revert InvalidOutputType(outputType);
		}
	}

	function _decodeVaaEssentials(bytes calldata data, uint256 envelopeOffset) internal pure returns (uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) {
		// NOTE: We can't use the VaaLib version of this because it checks the version field as well
		uint256 offset = envelopeOffset + VaaLib.ENVELOPE_EMITTER_CHAIN_ID_OFFSET;
		(emitterChainId, offset) = data.asUint16CdUnchecked(offset);
		(emitterAddress, offset) = data.asBytes32CdUnchecked(offset);
		(sequence,             ) = data.asUint32CdUnchecked(offset);

		uint payloadOffset = envelopeOffset + VaaLib.ENVELOPE_SIZE;
		payload = data.decodeVaaPayloadCd(payloadOffset);
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

	function update(bytes calldata data) public {
		(uint8 opcode,) = data.asUint8CdUnchecked(0);

		if (opcode == 0) {
			_updateShardId(data, 1);
		} else if (opcode == 1) {
			_appendSchnorrKeys(data, 1);
		} else if (opcode == 2) {
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
				VerificationError errorCode = _verifyMultisig(vaaDoubleHash, guardians, signatureCount, data, signaturesOffset);
				require(errorCode == VerificationError.NoError, FailedVerification(errorCode));

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
