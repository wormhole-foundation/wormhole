// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import {CoreBridgeLib} from "wormhole-sdk/libraries/CoreBridge.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";

import {MultisigVerificationState} from "./MultisigVerificationState.sol";

contract MultisigVerification is MultisigVerificationState {
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using UncheckedIndexing for address[];

  error QuorumNotMet();
  error MultisigSignatureVerificationFailed();
  error GuardianSetExpired();

  constructor(
    address coreBridge,
    uint256 initGuardianSetIndex,
    uint256 pullLimit
  ) MultisigVerificationState(coreBridge, initGuardianSetIndex, pullLimit) {}

  function _verifyMultisigVaaHeader(bytes calldata encodedVaa) internal view returns (
    uint envelopeOffset,
    uint32 guardianSetIndex,
    address[] memory guardians
  ) {
    unchecked {
      uint offset = 0;
      uint8 version;
      uint signatureCount;

      (version, offset) = encodedVaa.asUint8CdUnchecked(offset);
      (guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
      (signatureCount, offset) = encodedVaa.asUint8CdUnchecked(offset);

      // Validate the version
      require(version == 1, VaaLib.InvalidVersion(version));

      // Get the guardian set and validate it's not expired
      uint32 expirationTime;
      (expirationTime, guardians) = _getGuardianSetInfo(guardianSetIndex);
      require(expirationTime > block.timestamp, GuardianSetExpired());

      // Get the number of signatures
      // NOTE: Optimization puts guardianCount on stack thus avoids mloads
      uint guardianCount = guardians.length;

      // Validate the number of signatures
      // NOTE: This works for empty guardian sets, because the quorum when there
      // are no guardians is 1
      uint quorumCount = CoreBridgeLib.minSigsForQuorum(guardianCount);
      require(signatureCount >= quorumCount, QuorumNotMet());

      // Calculate envelope offset and VAA hash
      envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);

      // Verify the signatures
      // NOTE: Optimization instead of always checking i == 0
      bool isFirstSignature = true;
      uint prevGuardian;
      
      for (uint i = 0; i < signatureCount; ++i) {
        // Decode the guardian index, r, s, and v
        uint guardian; bytes32 r; bytes32 s; uint8 v;
        (guardian, r, s, v, offset) = encodedVaa.decodeGuardianSignatureCdUnchecked(offset);

        // Verify the signature
        address signatory = ecrecover(vaaHash, v, r, s);
        address guardianAddress = guardians.readUnchecked(guardian);

        // Check that:
        // * the guardian indices are in strictly ascending order (only after the first signature)
        //     this is itself an optimization to efficiently prevent having the same guardian signature
        //     included twice
        // * that the guardian index is not out of bounds
        // * that the signatory is the guardian
        //
        // The core bridge also includes a separate check that signatory is not the zero address
        //   but this is already covered by comparing that the signatory matches the guardian which
        //   [can never be the zero address](https://github.com/wormhole-foundation/wormhole/blob/1dbe8459b96e182932d0dd5ae4b6bbce6f48cb09/ethereum/contracts/Setters.sol#L20)
        bool failed = eagerOr(
          eagerOr(
            !eagerOr(isFirstSignature, guardian > prevGuardian),
            guardian >= guardianCount
          ),
          signatory != guardianAddress
        );
        
        require(!failed, MultisigSignatureVerificationFailed());

        prevGuardian = guardian;
        isFirstSignature = false;
      }
    }
  }

  // Verify a guardian set VAA
  function _verifyAndDecodeMultisigVaa(bytes calldata encodedVaa) internal view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes calldata payload,
    uint32 guardianSetIndex,
    address[] memory guardians
  ) {
    uint payloadOffset;
    (payloadOffset, guardianSetIndex, guardians) = _verifyMultisigVaaHeader(encodedVaa);
  
    (
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      sequence,
      consistencyLevel,
      payload
    ) = encodedVaa.decodeVaaBodyCd(payloadOffset);
  }
}
