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

	function _setSchnorrShardId(uint40 shardBase, uint8 shardIndex, bytes32 id) internal {
		_schnorrShardData[shardBase + shardIndex].id = id;
	}
}

uint8 constant VERIFY_VAA_INPUT_RAW = 0;
uint8 constant VERIFY_VAA_INPUT_HASH_AND_HEADER = 1;
uint8 constant VERIFY_VAA_INPUT_HASH_AND_SCHNORR_SIGNATURE = 2;
uint8 constant VERIFY_VAA_INPUT_HASH_AND_MULTISIG_SIGNATURE = 3;

uint8 constant VERIFY_VAA_OUTPUT_REVERT = 0 << 4;
uint8 constant VERIFY_VAA_OUTPUT_BOOL = 1 << 4;
uint8 constant VERIFY_VAA_OUTPUT_ERROR_CODE = 2 << 4;
uint8 constant VERIFY_VAA_OUTPUT_BODY = 3 << 4; // NOTE: Only valid for INPUT_RAW
uint8 constant VERIFY_VAA_OUTPUT_ESSENTIALS = 4 << 4; // NOTE: Only valid for INPUT_RAW

abstract contract VerificationOptions {
	error InvalidInputType(uint8 inputType);
	error InvalidOutputType(uint8 outputType);

	uint8 constant private VERIFY_VAA_INPUT_MASK = 0x0F;
	uint8 constant private VERIFY_VAA_OUTPUT_MASK = 0xF0;

	uint256 constant internal VAA_COMMON_HEADER_LENGTH = 1 + 4;
	uint256 constant internal SCHNORR_SIGNATURE_LENGTH = 20 + 32;
	uint256 constant internal SCHNORR_VAA_HEADER_LENGTH = VAA_COMMON_HEADER_LENGTH + SCHNORR_SIGNATURE_LENGTH;

	function _decodeOptions(uint8 options) internal pure returns (uint8 inputType, uint8 outputType) {
		inputType = options & VERIFY_VAA_INPUT_MASK;
		outputType = options & VERIFY_VAA_OUTPUT_MASK;
	}
}

abstract contract VerificationSingle is VerificationOptions, VerificationStateMultisig, VerificationStateSchnorr, VerificationCore {
	using BytesParsing for bytes;
	using VaaLib for bytes;

	function verifyVaa(uint8 options, bytes calldata data) public view returns (bytes memory) {
		unchecked {
			(uint8 inputType, uint8 outputType) = _decodeOptions(options);

			// TODO: Binary tree?
			if (inputType == VERIFY_VAA_INPUT_RAW) {
				return _verifyVaaRaw(outputType, data);
			} else if (inputType == VERIFY_VAA_INPUT_HASH_AND_HEADER) {
				return _verifyVaaHashAndHeader(outputType, data);
			} else if (inputType == VERIFY_VAA_INPUT_HASH_AND_SCHNORR_SIGNATURE) {
				return _verifyVaaHashAndSchnorr(outputType, data);
			} else if (inputType == VERIFY_VAA_INPUT_HASH_AND_MULTISIG_SIGNATURE) {
				return _verifyVaaHashAndMultisig(outputType, data);
			} else {
				revert InvalidInputType(inputType);
			}
		}
	}

	function _verifyVaaRaw(uint8 outputType, bytes calldata data) internal view returns (bytes memory) {
		unchecked {
			uint256 offset = 0;
			uint8 version;
			uint32 keyIndex;

			(version, offset) = data.asUint8CdUnchecked(offset);
			(keyIndex, offset) = data.asUint32CdUnchecked(offset);

			if (version == 2) {
				return _verifyVaaSchnorr(outputType, keyIndex, data, offset);
			} else if (version == 1) {
				return _verifyVaaMultisig(outputType, keyIndex, data, offset);
			} else {
				return _generateOutputABI(outputType, VerificationError.InvalidVAAVersion, data, offset);
			}
		}
	}

	function _verifyVaaHashAndHeader(uint8 outputType, bytes calldata data) internal view returns (bytes memory) {
		unchecked {
			uint256 offset = 0;
			bytes32 vaaHash;
			uint8 version;
			uint32 keyIndex;

			(vaaHash, offset) = data.asBytes32CdUnchecked(offset);
			(version, offset) = data.asUint8CdUnchecked(offset);
			(keyIndex, offset) = data.asUint32CdUnchecked(offset);

			if (version == 2) {
				return _verifyVaaSchnorr(outputType, keyIndex, data, offset);
			} else if (version == 1) {
				return _verifyVaaMultisig(outputType, keyIndex, data, offset);
			} else {
				return _generateOutputABI(outputType, VerificationError.InvalidVAAVersion, data, offset);
			}
		}
	}

	function _verifyVaaHashAndSchnorr(uint8 outputType, bytes calldata data) internal pure returns (bytes memory) {
		uint256 offset = 0;
		bytes32 vaaHash;
		uint256 pubkey;

		(vaaHash, offset) = data.asBytes32CdUnchecked(offset);
		(pubkey, offset) = data.asUint256CdUnchecked(offset);

		VerificationError errorCode = _verifySchnorr(vaaHash, pubkey, data, offset);
		return _generateOutputABI(outputType, errorCode, data, offset);
	}

	function _verifyVaaHashAndMultisig(uint8 outputType, bytes calldata data) internal pure returns (bytes memory) {
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
		return _generateOutputABI(outputType, errorCode, data, offset);
	}

	function _verifyVaaSchnorr(uint8 outputType, uint32 keyIndex, bytes calldata data, uint256 offset) internal view returns (bytes memory) {
		SchnorrKeyData memory keyData = _getSchnorrKeyData(keyIndex);
		uint32 expirationTime = keyData.expirationTime;
		if (eagerAnd(expirationTime != 0, expirationTime < block.timestamp)) {
			return _generateOutputABI(outputType, VerificationError.KeyDataExpired, data, offset);
		}

		bytes32 vaaHash = data.calcVaaDoubleHashCd(SCHNORR_VAA_HEADER_LENGTH);
		VerificationError errorCode = _verifySchnorr(vaaHash, keyData.pubkey, data, offset);
		return _generateOutputABI(outputType, errorCode, data, offset);
	}

	function _verifyVaaMultisig(uint8 outputType, uint32 keyIndex, bytes calldata data, uint256 offset) internal view returns (bytes memory) {
		uint8 signatureCount;

		(signatureCount, offset) = data.asUint8CdUnchecked(offset);

		// Get the guardian set and validate it's not expired
		(uint32 expirationTime, address[] memory keys) = _getMultisigKeyData(keyIndex);
		if (eagerAnd(expirationTime != 0, expirationTime < block.timestamp)) {
			return _generateOutputABI(outputType, VerificationError.KeyDataExpired, data, offset);
		}

		uint256 envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
		bytes32 vaaHash = data.calcVaaDoubleHashCd(envelopeOffset);
		VerificationError errorCode = _verifyMultisig(vaaHash, keys, signatureCount, data, offset);
		return _generateOutputABI(outputType, errorCode, data, offset);
	}

	function _generateOutputABI(uint8 outputType, VerificationError errorCode, bytes calldata vaa, uint256 envelopeOffset) internal pure returns (bytes memory) {
		unchecked {
			if (outputType == VERIFY_VAA_OUTPUT_ESSENTIALS) {
				require(errorCode == VerificationError.NoError, FailedVerification(errorCode));

				// FIXME: This will return junk data if inputType is not valid!
				uint16 emitterChainId;
				bytes32 emitterAddress;
				uint64 sequence;

				uint256 essentialsOffset = envelopeOffset + VaaLib.ENVELOPE_EMITTER_CHAIN_ID_OFFSET;
				(emitterChainId, essentialsOffset) = vaa.asUint16CdUnchecked(essentialsOffset);
				(emitterAddress, essentialsOffset) = vaa.asBytes32CdUnchecked(essentialsOffset);
				(sequence,                       ) = vaa.asUint64CdUnchecked(essentialsOffset);

				uint payloadOffset = envelopeOffset + VaaLib.ENVELOPE_SIZE;
				bytes calldata payload = vaa.decodeVaaPayloadCd(payloadOffset);

				return abi.encode(emitterChainId, emitterAddress, sequence, payload);
			} else if (outputType == VERIFY_VAA_OUTPUT_REVERT) {
				require(errorCode == VerificationError.NoError, FailedVerification(errorCode));
				return new bytes(0);
			} else if (outputType == VERIFY_VAA_OUTPUT_ERROR_CODE) {
				bytes memory result = new bytes(1);
				result[0] = bytes1(uint8(errorCode));
				return result;
			} else if (outputType == VERIFY_VAA_OUTPUT_BODY) {
				require(errorCode == VerificationError.NoError, FailedVerification(errorCode));

				// FIXME: This will return junk data if inputType is not valid!
				return abi.encode(vaa.decodeVaaBodyStructCd(envelopeOffset));
			} else {
				revert InvalidOutputType(outputType);
			}
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

contract Verification is VerificationSingle, EIP712Encoding {
	using BytesParsing for bytes;
	using VaaLib for bytes;
	using UncheckedIndexing for address[];
	using {BytesParsing.checkLength} for uint;

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
		pullMultisigKeyData(pullLimit);
	}

	function getCurrentSchnorrKeyData() public view returns (uint32 index, uint256 pubkey) {
		unchecked {
			index = uint32(_getSchnorrKeyCount() - 1);
			pubkey = _getSchnorrKeyData(index).pubkey;
		}
	}

	function getSchnorrKeyData(uint32 index) public view returns (uint32 expirationTime, uint256 pubkey) {
		unchecked {
			SchnorrKeyData memory keyData = _getSchnorrKeyData(index);
			expirationTime = keyData.expirationTime;
			pubkey = keyData.pubkey;
		}
	}

	function getCurrentMultisigKeyData() public view returns (uint32 index, address[] memory guardians) {
		unchecked {
			index = uint32(_getMultisigKeyCount() - 1);
			(, guardians) = _getMultisigKeyData(index);
		}
	}

	function getMultisigKeyData(uint32 index) public view returns (uint32 expirationTime, address[] memory guardians) {
		unchecked {
			(expirationTime, guardians) = _getMultisigKeyData(index);
		}
	}

	function updateShardId(bytes calldata message) public {
		uint256 offset = 0;
		uint32 schnorrKeyIndex;
		uint32 expirationTime;
		bytes32 guardianId;
		uint8 guardianIndex;
		bytes32 r;
		bytes32 s;
		uint8 v;

		(schnorrKeyIndex, offset) = message.asUint32CdUnchecked(offset);
		(expirationTime, offset) = message.asUint32CdUnchecked(offset);
		(guardianId, offset) = message.asBytes32CdUnchecked(offset);
		(guardianIndex, r, s, v, offset) = message.decodeGuardianSignatureCdUnchecked(offset);

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
		require(signatory == keys.readUnchecked(guardianIndex), FailedVerification(VerificationError.SignatureMismatch));

		// Store the shard ID
		require(guardianIndex < keyData.shardCount, InvalidGuardianIndex());
		_setSchnorrShardId(keyData.shardBase, guardianIndex, guardianId);
	}

	function appendSchnorrKeys(bytes[] calldata encodedVaas) public {
		unchecked {
			// Decode the VAAs
			for (uint256 i = 0; i < encodedVaas.length; i++) {
				bytes calldata encodedVaa = encodedVaas[i];

				uint256 offset = 0;
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

				(version, offset) = encodedVaa.asUint8CdUnchecked(offset);
				(multisigKeyIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
				(signatureCount, offset) = encodedVaa.asUint8CdUnchecked(offset);

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
				) = encodedVaa.decodeVaaEnvelopeCdUnchecked(envelopeOffset);

				(module, offset) = encodedVaa.asBytes32MemUnchecked(offset);
				(action, offset) = encodedVaa.asUint8MemUnchecked(offset);

				(newSchnorrKeyIndex, offset) = encodedVaa.asUint32MemUnchecked(offset);
				(newSchnorrKey, offset) = encodedVaa.asUint256MemUnchecked(offset);
				(expirationDelaySeconds, offset) = encodedVaa.asUint32MemUnchecked(offset);

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
				bytes32 vaaDoubleHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);
				VerificationError errorCode = _verifyMultisig(vaaDoubleHash, guardians, signatureCount, encodedVaa, signaturesOffset);
				require(errorCode == VerificationError.NoError, FailedVerification(errorCode));

				// If there is a previous schnorr key that is now expired, store the expiration time
				if (newSchnorrKeyIndex > 0) {
					uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
					_setSchnorrExpirationTime(newSchnorrKeyIndex - 1, expirationTime);
				}

				// Store the new schnorr key data
				offset = _appendSchnorrKeyData(newSchnorrKey, multisigKeyIndex, signatureCount, encodedVaa, offset);

				// Ensure the VAA is fully consumed
				encodedVaa.length.checkLength(offset);
			}
		}
	}

	function pullMultisigKeyData(uint32 limit) public {
		unchecked {
			uint256 currentMultisigKeyIndex = _coreBridge.getCurrentGuardianSetIndex();
			uint256 currentMultisigKeysLength = currentMultisigKeyIndex + 1;
			uint256 oldMultisigKeysLength = _getMultisigKeyCount();

			if (currentMultisigKeysLength == oldMultisigKeysLength) return;

			// Check if we need to update the current guardian set
      if (oldMultisigKeysLength > 0) {
        // Pull and write the current guardian set expiration time
        uint32 updateIndex = uint32(oldMultisigKeysLength - 1);
        (, uint32 expirationTime) = _pullMultisigKeyData(updateIndex);
        _setMultisigExpirationTime(updateIndex, expirationTime);
      }

			// Calculate the upper bound of the guardian sets to pull
      uint upper = eagerOr(limit == 0, currentMultisigKeysLength - oldMultisigKeysLength < limit)
        ? currentMultisigKeysLength : oldMultisigKeysLength + limit;

      // Pull and append the guardian sets
      for (uint i = oldMultisigKeysLength; i < upper; i++) {
        // Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
        (bytes memory data, uint32 expirationTime) = _pullMultisigKeyData(uint32(i));
        _appendMultisigKeyData(data, expirationTime);
      }
		}
	}

	function _pullMultisigKeyData(uint32 index) private view returns (
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
