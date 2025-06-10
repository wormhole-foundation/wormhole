// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";

import {eagerAnd, eagerOr} from "wormhole-solidity-sdk/Utils.sol";
import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {ICoreBridge, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {CoreBridgeLib} from "wormhole-solidity-sdk/libraries/CoreBridge.sol";
import {UncheckedIndexing} from "wormhole-solidity-sdk/libraries/UncheckedIndexing.sol";
import {VaaLib} from "wormhole-solidity-sdk/libraries/VaaLib.sol";

import {ExtStore} from "./ExtStore.sol";

uint256 constant OP_EXEC_APPEND_SCHNORR_KEY = 0;
uint256 constant OP_EXEC_PULL_MULTISIG_KEYS = 1;
uint256 constant OP_EXEC_SET_SCHNORR_SHARD_ID = 2;

uint256 constant OP_GET_VERIFY_VAA = 0;
uint256 constant OP_GET_MAYBE_VERIFY_VAA = 1;

// Governance emitter address
bytes32 constant GOVERNANCE_ADDRESS = bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);

// Module ID for the VerificationV2 contract, ASCII "TSS"
bytes32 constant MODULE_VERIFICATION_V2 = bytes32(0x0000000000000000000000000000000000000000000000000000000000545353);

// Action ID for appending a threshold key
uint8 constant ACTION_APPEND_SCHNORR_KEY = 0x01;

contract Optimized is ExtStore {
	using BytesParsing for bytes;
	using VaaLib for bytes;
	using UncheckedIndexing for address[];
	using {BytesParsing.checkLength} for uint;

	error InvalidPayment();
	error InvalidOperation(uint256 offset, uint8 op);

	error FailedVerification(VerificationError errorCode); // TODO: Replace this with normal errors

	struct SchnorrKeyData {
		uint256 pubkey;
		uint32 expirationTime;
		uint8 shardCount;
		uint40 shardBase;
		uint32 multisigKeyIndex;
	}

	struct ShardInfo {
		bytes32 shard;
		bytes32 id;
	}
	
	enum VerificationError {
		NoError, // 0
		InvalidVAAVersion, // 1
		InvalidMultisigKeyIndex, // 2
		InvalidSchnorrKeyIndex, // 3
		InvalidSchnorrKey, // 4
		InvalidGovernanceChain, // 5
		InvalidGovernanceAddress, // 6
		InvalidModule, // 7
		InvalidAction, // 8
		SignatureMismatch, // 9
		InvalidSignature, // 10
		KeyDataExpired, // 11
		QuorumNotMet // 12
	}

	// Curve order for secp256k1
  uint256 constant internal Q = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
  uint256 constant internal HALF_Q = Q >> 1;
	uint256 constant internal SCHNORR_VAA_HEADER_LENGTH = 1 + 4 + 20 + 32;

	ICoreBridge private immutable _coreBridge;
	uint32[] private _guardianSetExpirationTimes;

	SchnorrKeyData[] private _schnorrKeyData;
	ShardInfo[] private _schnorrShardData;

	constructor(ICoreBridge coreBridge, uint32 initMultisigKeyIndex, uint32 pullLimit) {
		_coreBridge = coreBridge;

		require(initMultisigKeyIndex <= _coreBridge.getCurrentGuardianSetIndex());
		// All previous guardian sets will have expiration timestamp 0
		// FIXME: Is this okay? It seems like it would validate VAAs from before the initMultisigKeyIndex
		assembly ("memory-safe") {
			sstore(_guardianSetExpirationTimes.slot, initMultisigKeyIndex)
		}

		_pullGuardianSets(pullLimit);
	}

	function verifyVaa(bytes calldata vaa) external view {
		uint8 version = uint8(vaa[0]);
		uint256 envelopeOffset = version == 2 ? SCHNORR_VAA_HEADER_LENGTH : VaaLib.skipVaaHeaderCdUnchecked(vaa);
		_verifyVaaHeader(vaa.calcVaaDoubleHashCd(envelopeOffset), vaa);
	}

	function exec768() external payable returns (bytes memory) {
		unchecked {
			require(msg.value == 0, InvalidPayment());
			uint256 offset = 4;

			while (offset < msg.data.length) {
				uint8 op;
				(op, offset) = msg.data.asUint8CdUnchecked(offset);

				if (op == OP_EXEC_APPEND_SCHNORR_KEY) {
					bytes calldata encodedVaa;
					(encodedVaa, offset) = msg.data.sliceUint16PrefixedCdUnchecked(offset);

					_appendSchnorrKey(encodedVaa);
				} else if (op == OP_EXEC_PULL_MULTISIG_KEYS) {
					uint32 limit;
					(limit, offset) = msg.data.asUint32CdUnchecked(offset);

					_pullGuardianSets(limit);
				} else if (op == OP_EXEC_SET_SCHNORR_SHARD_ID) {

				} else {
					revert InvalidOperation(offset, op);
				}
			}

			// Check if there is any data left
			msg.data.length.checkLength(offset);

			return new bytes(0);
		}
	}

	function get1959() external view returns (bytes memory) {
		unchecked {
			uint256 offset = 4;

			bytes memory result;

			while (offset < msg.data.length) {
				uint8 op;
				(op, offset) = msg.data.asUint8CdUnchecked(offset);

				if (op == OP_GET_VERIFY_VAA) {
					bytes32 vaaDoubleHash;
					bytes calldata header;

					(vaaDoubleHash, offset) = msg.data.asBytes32CdUnchecked(offset);
					(header, offset) = msg.data.sliceUint16PrefixedCdUnchecked(offset);

					VerificationError errorCode = _verifyVaaHeader(vaaDoubleHash, header);
					if (errorCode != VerificationError.NoError) {
						revert FailedVerification(errorCode);
					}
				} else if (op == OP_GET_MAYBE_VERIFY_VAA) {
					bytes32 vaaDoubleHash;
					bytes calldata header;

					(vaaDoubleHash, offset) = msg.data.asBytes32CdUnchecked(offset);
					(header, offset) = msg.data.sliceUint16PrefixedCdUnchecked(offset);
					VerificationError errorCode = _verifyVaaHeader(vaaDoubleHash, header);
					result = abi.encodePacked(result, errorCode);
				} else {
					revert InvalidOperation(offset, op);
				}
			}

			// Check if there is any data left
			msg.data.length.checkLength(offset);

			return result;
		}
	}

	function _verifyMultisig(
		bytes32 vaaDoubleHash,
		address[] memory guardians,
		uint256 signatureCount,
		bytes calldata signatures,
		uint256 offset
	) internal pure returns (VerificationError errorCode) {
		unchecked {
			// Verify the signatures;
			uint256 guardianCount = guardians.length - 1;
      uint256 usedGuardianBitfield;
      
      for (uint256 i = 0; i < signatureCount; i++) {
        // Decode the guardian index, r, s, and v
        uint256 guardian; bytes32 r; bytes32 s; uint8 v;
        (guardian, r, s, v, offset) = signatures.decodeGuardianSignatureCdUnchecked(offset);

        // Verify the signature
        address signatory = ecrecover(vaaDoubleHash, v, r, s);
        address guardianAddress = guardians.readUnchecked(guardian);

        // Check that:
        // * no guardian indices are included twice
        // * no guardian indices are out of bounds
        // * the signatory is the guardian at the given index

				uint256 guardianFlag = 1 << guardian;

        bool failed = eagerOr(
          eagerOr(
            (usedGuardianBitfield & guardianFlag) != 0,
            guardian > guardianCount
          ),
          signatory != guardianAddress
        );
        
        if (failed) return VerificationError.SignatureMismatch;

        usedGuardianBitfield |= guardianFlag;
      }

			return VerificationError.NoError;
		}
	}

	function _decodePubkey(uint256 pubkey) internal pure returns (uint256 px, uint8 parity) {
		unchecked {
			parity = uint8((pubkey & 1) + VaaLib.SIGNATURE_RECOVERY_MAGIC);
			px = pubkey >> 1;
		}
	}

	function _verifySchnorr(
		bytes32 vaaDoubleHash,
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
      uint256 e = uint256(keccak256(abi.encodePacked(px, parity == 28, vaaDoubleHash, r)));

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

	function _verifyVaaHeader(bytes32 vaaDoubleHash, bytes calldata header) private view returns (VerificationError errorCode) {
		unchecked {
			uint256 offset = 0;
			uint8 version;
			uint32 keyIndex;

			(version, offset) = header.asUint8CdUnchecked(offset);
			(keyIndex, offset) = header.asUint32CdUnchecked(offset);

			if (version == 2) {
				// Get the schnorr key data
				SchnorrKeyData memory info = _schnorrKeyData[keyIndex];

				// Validate the expiration time
				uint32 expirationTime = info.expirationTime;
				if (eagerAnd(expirationTime != 0, expirationTime <= block.timestamp)) {
					return VerificationError.KeyDataExpired;
				}

				return _verifySchnorr(vaaDoubleHash, info.pubkey, header, offset);
			} else if (version == 1) {
				uint8 signatureCount;

				(signatureCount, offset) = header.asUint8CdUnchecked(offset);

				// Get the guardian set info
				(uint32 expirationTime, address[] memory guardians) = _getMultisigKeyInfo(keyIndex);
				if (eagerAnd(expirationTime != 0, expirationTime <= block.timestamp)) {
					return VerificationError.KeyDataExpired;
				}

				// Validate the number of signatures
				uint256 quorumCount = CoreBridgeLib.minSigsForQuorum(guardians.length);
				if (signatureCount < quorumCount) {
					return VerificationError.QuorumNotMet;
				}

				// Verify the signatures
				return _verifyMultisig(vaaDoubleHash, guardians, signatureCount, header, offset);
			} else {
				return VerificationError.InvalidVAAVersion;
			}		
		}
	}

	function _appendSchnorrKey(bytes calldata encodedVaa) private {
		unchecked {
			// Decode the VAA
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
			(, address[] memory guardians) = _getMultisigKeyInfo(currentMultisigKeyIndex);

			// Decode the pubkey
			(uint256 px,) = _decodePubkey(newSchnorrKey);

			// Verify the VAA
			require(version == 1, FailedVerification(VerificationError.InvalidVAAVersion));
			require(multisigKeyIndex == currentMultisigKeyIndex, FailedVerification(VerificationError.InvalidMultisigKeyIndex));
			require(signatureCount == guardians.length, FailedVerification(VerificationError.QuorumNotMet));

			require(emitterChainId == CHAIN_ID_SOLANA, FailedVerification(VerificationError.InvalidGovernanceChain));
			require(emitterAddress == GOVERNANCE_ADDRESS, FailedVerification(VerificationError.InvalidGovernanceAddress));

			require(module == MODULE_VERIFICATION_V2, FailedVerification(VerificationError.InvalidModule));
			require(action == ACTION_APPEND_SCHNORR_KEY, FailedVerification(VerificationError.InvalidAction));
			require(newSchnorrKeyIndex == _schnorrKeyData.length, FailedVerification(VerificationError.InvalidSchnorrKeyIndex));
			require(eagerAnd(px != 0, px <= HALF_Q), FailedVerification(VerificationError.InvalidSchnorrKey));

			// Verify the signatures
			bytes32 vaaDoubleHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);
			VerificationError errorCode = _verifyMultisig(vaaDoubleHash, guardians, signatureCount, encodedVaa, signaturesOffset);
			if (errorCode != VerificationError.NoError) {
				revert FailedVerification(errorCode);
			}

			// If there is a previous schnorr key that is now expired, store the expiration time
			if (newSchnorrKeyIndex > 0) {
				uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
				_schnorrKeyData[newSchnorrKeyIndex - 1].expirationTime = expirationTime;
			}

			// Store the new schnorr key data
			_schnorrKeyData.push(SchnorrKeyData({
				pubkey: newSchnorrKey,
				expirationTime: 0,
				shardCount: signatureCount,
				shardBase: uint40(_schnorrShardData.length),
				multisigKeyIndex: multisigKeyIndex
			}));

			// Store the shard data
			for (uint256 i = 0; i < signatureCount; i++) {
				bytes32 shard;
				bytes32 id;

				(shard, offset) = encodedVaa.asBytes32CdUnchecked(offset);
				(id, offset) = encodedVaa.asBytes32CdUnchecked(offset);

				_schnorrShardData.push(ShardInfo({
					shard: shard,
					id: id
				}));
			}

			// Ensure the VAA is fully consumed
			encodedVaa.length.checkLength(offset);
		}
	}

	function _getMultisigKeyInfo(uint32 multisigKeyIndex) private view returns (uint32 expirationTime, address[] memory guardians) {
		unchecked {
			expirationTime = _guardianSetExpirationTimes[multisigKeyIndex];
      bytes memory data = _extRead(multisigKeyIndex);

      // Convert the guardian set data to an array of addresses
      // NOTE: The `data` array is temporary and is invalid after this block
      assembly ("memory-safe") {
        guardians := data
        mstore(guardians, shr(5, mload(guardians)))
      }
		}
	}

	function _pullGuardianSets(uint32 limit) private {
		unchecked {
			uint256 currentMultisigKeyIndex = _coreBridge.getCurrentGuardianSetIndex();
			uint256 currentMultisigKeysLength = currentMultisigKeyIndex + 1;
			uint256 oldMultisigKeysLength = _guardianSetExpirationTimes.length;

			if (currentMultisigKeysLength == oldMultisigKeysLength) return;

			// Check if we need to update the current guardian set
      if (oldMultisigKeysLength > 0) {
        // Pull and write the current guardian set expiration time
        uint updateIndex = oldMultisigKeysLength - 1;
        (, uint32 expirationTime) = _pullGuardianSet(uint32(updateIndex));
        _guardianSetExpirationTimes[updateIndex] = expirationTime;
      }

			// Calculate the upper bound of the guardian sets to pull
      uint upper = eagerOr(limit == 0, currentMultisigKeysLength - oldMultisigKeysLength < limit)
        ? currentMultisigKeysLength : oldMultisigKeysLength + limit;

      // Pull and append the guardian sets
      for (uint i = oldMultisigKeysLength; i < upper; i++) {
        // Pull the guardian set, write the expiration time, and append the guardian set data to the ExtStore
        (bytes memory data, uint32 expirationTime) = _pullGuardianSet(uint32(i));
        _guardianSetExpirationTimes.push(expirationTime);
        _extWrite(data);
      }
		}
	}

	function _pullGuardianSet(uint32 index) private view returns (
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
