// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  InvalidPayloadId,
  InvalidPayloadLength,
  InvalidVaaKeyType,
  VaaKey,
  VaaKeyType,
  Send,
  ExecutionParameters,
  DeliveryInstruction,
  RedeliveryInstruction,
  DeliveryOverride
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {BytesParsing} from "./BytesParsing.sol";

library CoreRelayerSerde {
  using BytesParsing for bytes;

  // ---------------------- "public" (i.e implicitly internal) encode/decode -----------------------

  //The slightly subtle difference between `PAYLOAD_ID`s and `VERSION`s is that payload ids carry
  //  both type information _and_ version information, while `VERSION`s only carry the latter.
  //That is, when deserialing a "version struct" we already know the expected type, but since we
  //  publish both Delivery _and_ Redelivery instructions as serialized messages, we need a robust
  //  way to distinguish both their type and their version during deserialization.
  uint8 private constant VERSION_VAAKEY = 1;
  uint8 private constant VERSION_EXECUTION_PARAMETERS = 1;
  uint8 private constant VERSION_DELIVERY_OVERRIDE = 1;
  uint8 private constant PAYLOAD_ID_DELIVERY_INSTRUCTION = 1;
  uint8 private constant PAYLOAD_ID_REDELIVERY_INSTRUCTION = 2;

  //TODO GAS OPTIMIZATION: All the recursive abi.encodePacked calls in here are _insanely_ gas
  //    inefficient (unless the optimizer is smart enough to just concatenate them tail-recursion
  //    style which seems highly unlikely)

  function encode(
    Send memory strct
  ) internal pure returns (bytes memory encoded) {
    //Send has no payload id/versioning because it is only used internally and never emitted
    encoded = abi.encodePacked(
      strct.targetChainId,
      strct.targetAddress,
      strct.refundChainId,
      strct.refundAddress,
      strct.maxTransactionFee,
      strct.receiverValue,
      strct.relayProviderAddress,
      encodeVaaKeyArray(strct.vaaKeys),
      strct.consistencyLevel,
      encodePayload(strct.payload),
      encodePayload(strct.relayParameters)
    );
  }

  function decodeSend(
    bytes memory encoded
  ) internal pure returns (Send memory strct) {
    uint256 offset = 0;
    (strct.targetChainId,        offset) = encoded.asUint16Unchecked(offset);
    (strct.targetAddress,        offset) = encoded.asBytes32Unchecked(offset);
    (strct.refundChainId,        offset) = encoded.asUint16Unchecked(offset);
    (strct.refundAddress,        offset) = encoded.asBytes32Unchecked(offset);
    (strct.maxTransactionFee,    offset) = encoded.asUint256Unchecked(offset);
    (strct.receiverValue,        offset) = encoded.asUint256Unchecked(offset);
    (strct.relayProviderAddress, offset) = encoded.asAddressUnchecked(offset);
    (strct.vaaKeys,              offset) = decodeVaaKeyArray(encoded, offset);
    (strct.consistencyLevel,     offset) = encoded.asUint8Unchecked(offset);
    (strct.payload,              offset) = decodePayload(encoded, offset);
    (strct.relayParameters,      offset) = decodePayload(encoded, offset);
    checkLength(encoded, offset);
  }

  function encode(
    DeliveryInstruction memory strct
  ) internal pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(
      PAYLOAD_ID_DELIVERY_INSTRUCTION,
      strct.targetChainId,
      strct.targetAddress,
      strct.refundChainId,
      strct.refundAddress,
      strct.maximumRefundTarget,
      strct.receiverValueTarget,
      strct.sourceRelayProvider,
      strct.targetRelayProvider,
      strct.senderAddress,
      encodeVaaKeyArray(strct.vaaKeys),
      strct.consistencyLevel,
      encodeExecutionParameters(strct.executionParameters),
      encodePayload(strct.payload)
    );
  }

  function decodeDeliveryInstruction(
    bytes memory encoded
  ) internal pure returns (DeliveryInstruction memory strct) {
    uint offset = checkUint8(encoded, 0, PAYLOAD_ID_DELIVERY_INSTRUCTION);
    (strct.targetChainId,       offset) = encoded.asUint16Unchecked(offset);
    (strct.targetAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.refundChainId,       offset) = encoded.asUint16Unchecked(offset);
    (strct.refundAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.maximumRefundTarget, offset) = encoded.asUint256Unchecked(offset);
    (strct.receiverValueTarget, offset) = encoded.asUint256Unchecked(offset);
    (strct.sourceRelayProvider, offset) = encoded.asBytes32Unchecked(offset);
    (strct.targetRelayProvider, offset) = encoded.asBytes32Unchecked(offset);
    (strct.senderAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.vaaKeys,             offset) = decodeVaaKeyArray(encoded, offset);
    (strct.consistencyLevel,    offset) = encoded.asUint8Unchecked(offset);
    (strct.executionParameters, offset) = decodeExecutionParameters(encoded, offset);
    (strct.payload,             offset) = decodePayload(encoded, offset);
    checkLength(encoded, offset);
  }

  function encode(
    RedeliveryInstruction memory strct
  ) internal pure returns (bytes memory encoded) {
    bytes memory vaaKey = encodeVaaKey(strct.key);
    encoded = abi.encodePacked(
      PAYLOAD_ID_REDELIVERY_INSTRUCTION,
      vaaKey,
      strct.newMaximumRefundTarget,
      strct.newReceiverValueTarget,
      strct.sourceRelayProvider,
      strct.targetChainId,
      encodeExecutionParameters(strct.executionParameters)
    );
  }

  function decodeRedeliveryInstruction(
    bytes memory encoded
  ) internal pure returns (RedeliveryInstruction memory strct) {
    uint256 offset = checkUint8(encoded, 0 , PAYLOAD_ID_REDELIVERY_INSTRUCTION);
    (strct.key,                    offset) = decodeVaaKey(encoded, offset);
    (strct.newMaximumRefundTarget, offset) = encoded.asUint256Unchecked(offset);
    (strct.newReceiverValueTarget, offset) = encoded.asUint256Unchecked(offset);
    (strct.sourceRelayProvider,    offset) = encoded.asBytes32Unchecked(offset);
    (strct.targetChainId,          offset) = encoded.asUint16Unchecked(offset);
    (strct.executionParameters,    offset) = decodeExecutionParameters(encoded, offset);
    checkLength(encoded, offset);
  }

  function encode(
    DeliveryOverride memory strct
  ) internal pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(
      VERSION_DELIVERY_OVERRIDE,
      strct.gasLimit,
      strct.maximumRefund,
      strct.receiverValue,
      strct.redeliveryHash
    );
  }

  function decodeDeliveryOverride(
    bytes memory encoded
  ) internal pure returns (DeliveryOverride memory strct) {
    uint offset = checkUint8(encoded, 0, VERSION_DELIVERY_OVERRIDE);
    (strct.gasLimit,       offset) = encoded.asUint32Unchecked(offset);
    (strct.maximumRefund,  offset) = encoded.asUint256Unchecked(offset);
    (strct.receiverValue,  offset) = encoded.asUint256Unchecked(offset);
    (strct.redeliveryHash, offset) = encoded.asBytes32Unchecked(offset);
    checkLength(encoded, offset);
  }

  // ------------------------------------------ private --------------------------------------------

  function encodeVaaKeyArray(
    VaaKey[] memory vaaKeys
  ) private pure returns (bytes memory encoded) {
    assert(vaaKeys.length < type(uint8).max);
    encoded = abi.encodePacked(uint8(vaaKeys.length));
    for (uint i = 0; i < vaaKeys.length;) {
      encoded = abi.encodePacked(encoded, encodeVaaKey(vaaKeys[i]));
      unchecked{++i;}
    }
  }

  function decodeVaaKeyArray(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (VaaKey[] memory vaaKeys, uint offset) {
    uint8 vaaKeysLength;
    (vaaKeysLength, offset) = encoded.asUint8Unchecked(startOffset);
    vaaKeys = new VaaKey[](vaaKeysLength);
    for (uint i = 0; i < vaaKeys.length;) {
      (vaaKeys[i], offset) = decodeVaaKey(encoded, offset);
      unchecked{++i;}
    }
  }

  function encodeVaaKey(
    VaaKey memory vaaKey
  ) private pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(VERSION_VAAKEY, uint8(vaaKey.infoType));
    if (vaaKey.infoType == VaaKeyType.EMITTER_SEQUENCE)
      encoded = abi.encodePacked(encoded, vaaKey.chainId, vaaKey.emitterAddress, vaaKey.sequence);
    else //vaaKey.infoType == VaaKeyType.VAAHASH)
      encoded = abi.encodePacked(encoded, vaaKey.vaaHash);
  }

  function decodeVaaKey(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (VaaKey memory vaaKey, uint offset) {
    offset = checkUint8(encoded, startOffset, VERSION_VAAKEY);

    uint8 parsedVaaKeyType;
    (parsedVaaKeyType, offset) = encoded.asUint8Unchecked(offset);
    //Explicitly casting int to enum panics for invalid values
    //  (see https://docs.soliditylang.org/en/v0.8.19/types.html#enums)
    //We want to revert with our custom error, so we explicitly check ourselves and only perform the
    //  cast below once it is known to be safe.
    if (parsedVaaKeyType == uint8(VaaKeyType.EMITTER_SEQUENCE)) {
      (vaaKey.chainId,        offset) = encoded.asUint16Unchecked(offset);
      (vaaKey.emitterAddress, offset) = encoded.asBytes32Unchecked(offset);
      (vaaKey.sequence,       offset) = encoded.asUint64Unchecked(offset);
    }
    else if (parsedVaaKeyType == uint8(VaaKeyType.VAAHASH)) {
      (vaaKey.vaaHash, offset) = encoded.asBytes32Unchecked(offset);
    }
    else
      revert InvalidVaaKeyType(parsedVaaKeyType);

    vaaKey.infoType = VaaKeyType(parsedVaaKeyType);
  }

  function encodePayload(
    bytes memory payload
  ) private pure returns (bytes memory encoded) {
    //casting payload.length to uint32 is safe because you'll be hard-pressed to allocate 4 GB of
    //  EVM memory in a single transaction
    encoded = abi.encodePacked(uint32(payload.length), payload);
  }

  function decodePayload(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (bytes memory payload, uint offset) {
    uint32 payloadLength;
    (payloadLength, offset) = encoded.asUint32Unchecked(startOffset);
    (payload,       offset) = encoded.sliceUnchecked(offset, payloadLength);
  }

  function encodeExecutionParameters(
    ExecutionParameters memory strct
  ) private pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(VERSION_EXECUTION_PARAMETERS, strct.gasLimit);
  }

  function decodeExecutionParameters(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (ExecutionParameters memory strct, uint offset) {
    offset = checkUint8(encoded, startOffset, VERSION_EXECUTION_PARAMETERS);
    (strct.gasLimit, offset) = encoded.asUint32Unchecked(offset);
  }

  function checkUint8(
    bytes memory encoded,
    uint startOffset,
    uint8 expectedPayloadId
  ) private pure returns (uint offset) {
    uint8 parsedPayloadId;
    (parsedPayloadId, offset) = encoded.asUint8Unchecked(startOffset);
    if (parsedPayloadId != expectedPayloadId)
      revert InvalidPayloadId(parsedPayloadId, expectedPayloadId);
  }

  function checkLength(bytes memory encoded, uint expected) private pure {
    if (encoded.length != expected)
      revert InvalidPayloadLength(encoded.length, expected);
  }
}