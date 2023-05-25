// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {
  InvalidPayloadId,
  InvalidPayloadLength,
  InvalidVaaKeyType,
  VaaKey
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {
  DeliveryOverride,
  DeliveryInstruction,
  RedeliveryInstruction
} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {BytesParsing} from "../../libraries/relayer/BytesParsing.sol";
import "../../interfaces/relayer/TypedUnits.sol";

library CoreRelayerSerde {
  using BytesParsing for bytes;
  using WeiLib for Wei;
  using GasLib for Gas;

  //The slightly subtle difference between `PAYLOAD_ID`s and `VERSION`s is that payload ids carry
  //  both type information _and_ version information, while `VERSION`s only carry the latter.
  //That is, when deserialing a "version struct" we already know the expected type, but since we
  //  publish both Delivery _and_ Redelivery instructions as serialized messages, we need a robust
  //  way to distinguish both their type and their version during deserialization.
  uint8 private constant VERSION_VAAKEY = 1;
  uint8 private constant VERSION_DELIVERY_OVERRIDE = 1;
  uint8 private constant PAYLOAD_ID_DELIVERY_INSTRUCTION = 1;
  uint8 private constant PAYLOAD_ID_REDELIVERY_INSTRUCTION = 2;

  // ---------------------- "public" (i.e implicitly internal) encode/decode -----------------------

  //TODO GAS OPTIMIZATION: All the recursive abi.encodePacked calls in here are _insanely_ gas
  //    inefficient (unless the optimizer is smart enough to just concatenate them tail-recursion
  //    style which seems highly unlikely)

  function encode(
    DeliveryInstruction memory strct
  ) internal pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(
      PAYLOAD_ID_DELIVERY_INSTRUCTION,
      strct.targetChainId,
      strct.targetAddress,
      encodeBytes(strct.payload),
      strct.requestedReceiverValue,
      strct.extraReceiverValue);
    encoded = abi.encodePacked(
      encoded,
      encodeBytes(strct.encodedExecutionInfo),
      strct.refundChainId,
      strct.refundAddress,
      strct.refundRelayProvider,
      strct.sourceRelayProvider,
      strct.senderAddress,
      encodeVaaKeyArray(strct.vaaKeys)
    );
  }

  function decodeDeliveryInstruction(
    bytes memory encoded
  ) internal pure returns (DeliveryInstruction memory strct) {
    uint offset = checkUint8(encoded, 0, PAYLOAD_ID_DELIVERY_INSTRUCTION);

    uint256 requestedReceiverValue;
    uint256 extraReceiverValue;

    (strct.targetChainId,       offset) = encoded.asUint16Unchecked(offset);
    (strct.targetAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.payload,             offset) = decodeBytes(encoded, offset);
    (requestedReceiverValue,    offset) = encoded.asUint256Unchecked(offset);
    (extraReceiverValue,        offset) = encoded.asUint256Unchecked(offset);
    (strct.encodedExecutionInfo,     offset) = decodeBytes(encoded, offset);
    (strct.refundChainId,       offset) = encoded.asUint16Unchecked(offset);
    (strct.refundAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.refundRelayProvider, offset) = encoded.asBytes32Unchecked(offset);
    (strct.sourceRelayProvider, offset) = encoded.asBytes32Unchecked(offset);
    (strct.senderAddress,       offset) = encoded.asBytes32Unchecked(offset);
    (strct.vaaKeys,             offset) = decodeVaaKeyArray(encoded, offset);

    strct.requestedReceiverValue = Wei.wrap(requestedReceiverValue);
    strct.extraReceiverValue     = Wei.wrap(extraReceiverValue);

    checkLength(encoded, offset);
  }

  function encode(
    RedeliveryInstruction memory strct
  ) internal pure returns (bytes memory encoded) {
    bytes memory vaaKey = encodeVaaKey(strct.deliveryVaaKey);
    encoded = abi.encodePacked(
      PAYLOAD_ID_REDELIVERY_INSTRUCTION,
      vaaKey,
      strct.targetChainId,
      strct.newRequestedReceiverValue,
      encodeBytes(strct.newEncodedExecutionInfo),
      strct.newSourceRelayProvider,
      strct.newSenderAddress
    );
  }

  function decodeRedeliveryInstruction(
    bytes memory encoded
  ) internal pure returns (RedeliveryInstruction memory strct) {
    uint256 offset = checkUint8(encoded, 0 , PAYLOAD_ID_REDELIVERY_INSTRUCTION);

    uint256 newRequestedReceiverValue;

    (strct.deliveryVaaKey,         offset) = decodeVaaKey(encoded, offset);
    (strct.targetChainId,          offset) = encoded.asUint16Unchecked(offset);
    (newRequestedReceiverValue,    offset) = encoded.asUint256Unchecked(offset);
    (strct.newEncodedExecutionInfo, offset)    = decodeBytes(encoded, offset);
    (strct.newSourceRelayProvider, offset) = encoded.asBytes32Unchecked(offset);
    (strct.newSenderAddress,    offset)    = encoded.asBytes32Unchecked(offset);

    strct.newRequestedReceiverValue = Wei.wrap(newRequestedReceiverValue);

    checkLength(encoded, offset);
  }

  function encode(
    DeliveryOverride memory strct
  ) internal pure returns (bytes memory encoded) {
    encoded = abi.encodePacked(
      VERSION_DELIVERY_OVERRIDE,
      strct.newReceiverValue,
      encodeBytes(strct.newExecutionInfo),
      strct.redeliveryHash
    );
  }

  function decodeDeliveryOverride(
    bytes memory encoded
  ) internal pure returns (DeliveryOverride memory strct) {
    uint offset = checkUint8(encoded, 0, VERSION_DELIVERY_OVERRIDE);

    uint256 receiverValue;
    
    (receiverValue,                 offset) = encoded.asUint256Unchecked(offset);
    (strct.newExecutionInfo,      offset) = decodeBytes(encoded, offset);
    (strct.redeliveryHash, offset) = encoded.asBytes32Unchecked(offset);

    strct.newReceiverValue = Wei.wrap(receiverValue);

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
    encoded = abi.encodePacked(encoded, VERSION_VAAKEY, vaaKey.chainId, vaaKey.emitterAddress, vaaKey.sequence);
  }

  function decodeVaaKey(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (VaaKey memory vaaKey, uint offset) {
    offset = checkUint8(encoded, startOffset, VERSION_VAAKEY);
    (vaaKey.chainId,        offset) = encoded.asUint16Unchecked(offset);
    (vaaKey.emitterAddress, offset) = encoded.asBytes32Unchecked(offset);
    (vaaKey.sequence,       offset) = encoded.asUint64Unchecked(offset);
  }

  function encodeBytes(
    bytes memory payload
  ) private pure returns (bytes memory encoded) {
    //casting payload.length to uint32 is safe because you'll be hard-pressed to allocate 4 GB of
    //  EVM memory in a single transaction
    encoded = abi.encodePacked(uint32(payload.length), payload);
  }

  function decodeBytes(
    bytes memory encoded,
    uint startOffset
  ) private pure returns (bytes memory payload, uint offset) {
    uint32 payloadLength;
    (payloadLength, offset) = encoded.asUint32Unchecked(startOffset);
    (payload,       offset) = encoded.sliceUnchecked(offset, payloadLength);
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