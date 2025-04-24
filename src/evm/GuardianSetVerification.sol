// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import {CoreBridgeLib} from "wormhole-sdk/libraries/CoreBridge.sol";
import {UncheckedIndexing} from "wormhole-sdk/libraries/UncheckedIndexing.sol";
import "./GuardianSetVerificationState.sol";

contract GuardianSetVerification is GuardianSetVerificationState {
  using BytesParsing for bytes;
  using VaaLib for bytes;
  using UncheckedIndexing for address[];

  constructor(
    address coreBridge,
    uint32 pullLimit
  ) GuardianSetVerificationState(coreBridge, pullLimit) {}

  function _verifyGuardianSetVaaHeader(bytes calldata encodedVaa) internal view returns (uint payloadOffset) {
    unchecked {
      uint offset = 0;
      uint32 guardianSetIndex;
      uint signatureCount;

      (guardianSetIndex, offset) = encodedVaa.asUint32CdUnchecked(offset);
      (signatureCount, offset) = encodedVaa.asUint8CdUnchecked(offset);

      // Get the guardian set and the number of guardians
      (, address[] memory guardians) = this.getGuardianSetInfo(guardianSetIndex);

      // Get the number of signatures
      // NOTE: Optimization puts guardianCount on stack thus avoids mloads
      uint guardianCount = guardians.length;

      // Validate the number of signatures
      // NOTE: This works for empty guardian sets, because the quorum when there
      // are no guardians is 1
      uint quorumCount = CoreBridgeLib.minSigsForQuorum(guardianCount);
      require(signatureCount >= quorumCount, VerificationFailed());

      // Calculate envelope offset and VAA hash
      uint envelopeOffset = offset + signatureCount * VaaLib.GUARDIAN_SIGNATURE_SIZE;
      bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);

      // Verify the signatures
      // NOTE: Optimization instead of always checking i == 0
      bool isFirstSignature = true;
      uint prevGuardianIndex;
      
      for (uint i = 0; i < signatureCount; ++i) {
        // Decode the guardian index, r, s, and v
        uint guardianIndex; bytes32 r; bytes32 s; uint8 v;
        (guardianIndex, r, s, v, offset) = encodedVaa.decodeGuardianSignatureCdUnchecked(offset);

        // Verify the signature
        address signatory = ecrecover(vaaHash, v, r, s);
        address guardian = guardians.readUnchecked(guardianIndex);

        // Check that:
        // * the guardian indicies are in strictly ascending order (only after the first signature)
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
            !eagerOr(isFirstSignature, guardianIndex > prevGuardianIndex),
            guardianIndex >= guardianCount
          ),
          signatory != guardian
        );
        
        // Verify the signature
        require(!failed, VerificationFailed());

        prevGuardianIndex = guardianIndex;
        isFirstSignature = false;
      }

      return offset;
    }
  }

  // Verify a guardian set VAA
  function _verifyAndDecodeGuardianSetVaa(bytes calldata encodedVaa) internal view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes calldata payload
  ) {
    uint payloadOffset = _verifyGuardianSetVaaHeader(encodedVaa);
    return encodedVaa.decodeVaaBodyCd(payloadOffset);
  }
}
