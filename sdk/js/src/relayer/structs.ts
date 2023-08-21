import { BigNumber, ethers } from "ethers";
import { arrayify } from "ethers/lib/utils";

export enum RelayerPayloadId {
  Delivery = 1,
  Redelivery = 2,
}

export enum ExecutionInfoVersion {
  EVM_V1 = 0,
}

export enum DeliveryStatus {
  WaitingForVAA = "Waiting for VAA",
  PendingDelivery = "Pending Delivery",
  DeliverySuccess = "Delivery Success",
  ReceiverFailure = "Receiver Failure",
  ThisShouldNeverHappen = "This should never happen. Contact Support.",
  DeliveryDidntHappenWithinRange = "Delivery didn't happen within given block range",
}

export enum RefundStatus {
  RefundSent = "Refund Sent",
  RefundFail = "Refund Fail",
  CrossChainRefundSent = "Cross Chain Refund Sent",
  CrossChainRefundFailProviderNotSupported = "Cross Chain Refund Fail - Provider does not support the refund chain",
  CrossChainRefundFailNotEnough = "Cross Chain Refund Fail - Refund too low for cross chain refund",
}

export function parseRefundStatus(index: number) {
  return index === 0
    ? RefundStatus.RefundSent
    : index === 1
    ? RefundStatus.RefundFail
    : index === 2
    ? RefundStatus.CrossChainRefundSent
    : index === 3
    ? RefundStatus.CrossChainRefundFailProviderNotSupported
    : index === 4
    ? RefundStatus.CrossChainRefundFailNotEnough
    : RefundStatus.CrossChainRefundFailProviderNotSupported;
}

export interface MessageKey {
  version: number;
  vaaKey?: VaaKey;
  encodedKey?: Buffer;
}

export interface VaaKey {
  chainId: number;
  emitterAddress: Buffer;
  sequence: BigNumber;
}

export interface DeliveryInstruction {
  targetChainId: number;
  targetAddress: Buffer;
  payload: Buffer;
  requestedReceiverValue: BigNumber;
  extraReceiverValue: BigNumber;
  encodedExecutionInfo: Buffer;
  refundChainId: number;
  refundAddress: Buffer;
  refundDeliveryProvider: Buffer;
  sourceDeliveryProvider: Buffer;
  senderAddress: Buffer;
  messageKeys: MessageKey[];
}

export interface RedeliveryInstruction {
  deliveryVaaKey: VaaKey;
  targetChainId: number;
  newRequestedReceiverValue: BigNumber;
  newEncodedExecutionInfo: Buffer;
  newSourceDeliveryProvider: Buffer;
  newSenderAddress: Buffer;
}

type StringLeaves<Type> =
  | string
  | string[]
  | { [P in keyof Type]: StringLeaves<Type[P]> };

export type DeliveryInstructionPrintable = {
  [Property in keyof DeliveryInstruction]: StringLeaves<
    DeliveryInstruction[Property]
  >;
};

export type RedeliveryInstructionPrintable = {
  [Property in keyof RedeliveryInstruction]: StringLeaves<
    RedeliveryInstruction[Property]
  >;
};

export interface EVMExecutionInfoV1 {
  gasLimit: BigNumber;
  targetChainRefundPerGasUnused: BigNumber;
}

export enum VaaKeyType {
  EMITTER_SEQUENCE = 0,
  VAAHASH = 1,
}

export function parseWormholeRelayerPayloadType(
  stringPayload: string | Buffer | Uint8Array
): RelayerPayloadId {
  const payload =
    typeof stringPayload === "string" ? arrayify(stringPayload) : stringPayload;
  if (
    payload[0] != RelayerPayloadId.Delivery &&
    payload[0] != RelayerPayloadId.Redelivery
  ) {
    throw new Error("Unrecognized payload type " + payload[0]);
  }
  return payload[0];
}

export function createVaaKey(
  chainId: number,
  emitterAddress: Buffer,
  sequence: number | BigNumber
): VaaKey {
  return {
    chainId,
    emitterAddress,
    sequence: ethers.BigNumber.from(sequence),
  };
}

export function parseWormholeRelayerSend(bytes: Buffer): DeliveryInstruction {
  let idx = 0;
  const payloadId = bytes.readUInt8(idx);
  if (payloadId !== RelayerPayloadId.Delivery) {
    throw new Error(
      `Expected Delivery payload type (${RelayerPayloadId.Delivery}), found: ${payloadId}`
    );
  }
  idx += 1;

  const targetChainId = bytes.readUInt16BE(idx);
  idx += 2;
  const targetAddress = bytes.slice(idx, idx + 32);
  idx += 32;

  let payload: Buffer;
  [payload, idx] = parsePayload(bytes, idx);

  const requestedReceiverValue = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  const extraReceiverValue = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  let encodedExecutionInfo;
  [encodedExecutionInfo, idx] = parsePayload(bytes, idx);

  const refundChainId = bytes.readUInt16BE(idx);
  idx += 2;
  const refundAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const refundDeliveryProvider = bytes.slice(idx, idx + 32);
  idx += 32;
  const sourceDeliveryProvider = bytes.slice(idx, idx + 32);
  idx += 32;
  const senderAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const numMessages = bytes.readUInt8(idx);
  idx += 1;

  let messages = [] as MessageKey[];
  for (let i = 0; i < numMessages; ++i) {
    const res = parseMessageKey(bytes, idx);
    idx = res[1];
    messages.push(res[0]);
  }

  return {
    targetChainId,
    targetAddress,
    payload,
    requestedReceiverValue,
    extraReceiverValue,
    encodedExecutionInfo,
    refundChainId,
    refundAddress,
    refundDeliveryProvider,
    sourceDeliveryProvider,
    senderAddress,
    messageKeys: messages,
  };
}

function parsePayload(bytes: Buffer, idx: number): [Buffer, number] {
  const length = bytes.readUInt32BE(idx);
  idx += 4;
  const payload = bytes.slice(idx, idx + length);
  idx += length;
  return [payload, idx];
}

export function encodeVaaKey(key: VaaKey): string {
  return ethers.utils.solidityPack(
    ["uint8", "uint16", "bytes32", "uint64"],
    [1, key.chainId, key.emitterAddress, key.sequence]
  );
}

function parseVaaKey(bytes: Buffer, idx: number): [VaaKey, number] {
  const version = bytes.readUInt8(idx);
  idx += 1;

  const chainId = bytes.readUInt16BE(idx);
  idx += 2;
  const emitterAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const sequence = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 8)
  );
  idx += 8;
  return [
    {
      chainId,
      emitterAddress,
      sequence,
    },
    idx,
  ];
}

function parseMessageKey(bytes: Buffer, idx: number): [MessageKey, number] {
  const version = bytes.readUInt8(idx);
  idx += 1;
  let vaaKey;
  let encodedKey;
  if (version === 1) {
    let oldIdx = idx;
    [vaaKey, idx] = parseVaaKey(bytes, idx);
    encodedKey = bytes.slice(oldIdx, idx);
  } else {
    const messageKeyEncodedLength = bytes.readUInt32BE(idx);
    idx += 4;
    const messageKeyEncoded: Buffer = bytes.slice(
      idx,
      idx + messageKeyEncodedLength
    );
    idx += messageKeyEncodedLength;

    encodedKey = messageKeyEncoded;
  }

  return [{ version, vaaKey, encodedKey }, idx];
}

export function parseEVMExecutionInfoV1(
  bytes: Buffer,
  idx: number
): [EVMExecutionInfoV1, number] {
  idx += 31;
  const version = bytes.readUInt8(idx);
  idx += 1;
  if (version !== ExecutionInfoVersion.EVM_V1) {
    throw new Error("Unexpected Execution Info version");
  }
  const gasLimit = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;
  const targetChainRefundPerGasUnused = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;
  return [{ gasLimit, targetChainRefundPerGasUnused }, idx];
}

export function parseWormholeRelayerResend(
  bytes: Buffer
): RedeliveryInstruction {
  let idx = 0;
  const payloadId = bytes.readUInt8(idx);
  if (payloadId !== RelayerPayloadId.Redelivery) {
    throw new Error(
      `Expected Delivery payload type (${RelayerPayloadId.Redelivery}), found: ${payloadId}`
    );
  }
  idx += 1;

  const parsedKey = parseVaaKey(bytes, idx);
  const key = parsedKey[0];
  idx = parsedKey[1];

  const targetChainId: number = bytes.readUInt16BE(idx);
  idx += 2;

  const newRequestedReceiverValue = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  let newEncodedExecutionInfo;
  [newEncodedExecutionInfo, idx] = parsePayload(bytes, idx);

  const newSourceDeliveryProvider = bytes.slice(idx, idx + 32);
  idx += 32;

  const newSenderAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  return {
    deliveryVaaKey: key,
    targetChainId,
    newRequestedReceiverValue,
    newEncodedExecutionInfo,
    newSourceDeliveryProvider,
    newSenderAddress,
  };
}

export function executionInfoToString(encodedExecutionInfo: Buffer): string {
  const [parsed] = parseEVMExecutionInfoV1(encodedExecutionInfo, 0);
  return `Gas limit: ${parsed.gasLimit}, Target chain refund per unit gas unused: ${parsed.targetChainRefundPerGasUnused}`;
}

export function deliveryInstructionsPrintable(
  ix: DeliveryInstruction
): DeliveryInstructionPrintable {
  return {
    targetChainId: ix.targetChainId.toString(),
    targetAddress: ix.targetAddress.toString("hex"),
    payload: ix.payload.toString("base64"),
    requestedReceiverValue: ix.requestedReceiverValue.toString(),
    extraReceiverValue: ix.requestedReceiverValue.toString(),
    encodedExecutionInfo: executionInfoToString(ix.encodedExecutionInfo),
    refundChainId: ix.refundChainId.toString(),
    refundAddress: ix.refundAddress.toString("hex"),
    refundDeliveryProvider: ix.refundDeliveryProvider.toString("hex"),
    sourceDeliveryProvider: ix.sourceDeliveryProvider.toString("hex"),
    senderAddress: ix.senderAddress.toString("hex"),
    messageKeys: ix.messageKeys.map(messageKeyPrintable),
  };
}
export function messageKeyPrintable(ix: MessageKey): StringLeaves<MessageKey> {
  return {
    version: ix.version,
    encodedKey: ix.encodedKey ? ix.encodedKey.toString("hex") : undefined,
    vaaKey: ix.vaaKey ? vaaKeyPrintable(ix.vaaKey) : undefined,
  };
}
export function vaaKeyPrintable(ix: VaaKey): StringLeaves<VaaKey> {
  return {
    chainId: ix.chainId?.toString(),
    emitterAddress: ix.emitterAddress?.toString("hex"),
    sequence: ix.sequence?.toString(),
  };
}

export function redeliveryInstructionPrintable(
  ix: RedeliveryInstruction
): RedeliveryInstructionPrintable {
  return {
    deliveryVaaKey: vaaKeyPrintable(ix.deliveryVaaKey),
    targetChainId: ix.targetChainId.toString(),
    newRequestedReceiverValue: ix.newRequestedReceiverValue.toString(),
    newEncodedExecutionInfo: executionInfoToString(ix.newEncodedExecutionInfo),
    newSourceDeliveryProvider: ix.newSourceDeliveryProvider.toString("hex"),
    newSenderAddress: ix.newSenderAddress.toString("hex"),
  };
}

export type DeliveryOverrideArgs = {
  newReceiverValue: BigNumber;
  newExecutionInfo: Buffer;
  redeliveryHash: Buffer;
};

export function packOverrides(overrides: DeliveryOverrideArgs): string {
  const packed = [
    ethers.utils.solidityPack(["uint8"], [1]).substring(2), //version
    ethers.utils
      .solidityPack(["uint256"], [overrides.newReceiverValue])
      .substring(2),
    ethers.utils
      .solidityPack(["uint32"], [overrides.newExecutionInfo.length])
      .substring(2),
    overrides.newExecutionInfo.toString("hex"),
    overrides.redeliveryHash.toString("hex"), //toString('hex') doesn't add the 0x prefix
  ].join("");

  return "0x" + packed;
}

export function parseOverrideInfoFromDeliveryEvent(
  bytes: Buffer
): DeliveryOverrideArgs {
  let idx = 0;
  const version = bytes.readUInt8(idx);
  idx += 1;
  const newReceiverValue = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  let newExecutionInfo: Buffer;
  [newExecutionInfo, idx] = parsePayload(bytes, idx);

  const redeliveryHash = bytes.slice(idx, idx + 32);
  idx += 32;

  return {
    newReceiverValue,
    newExecutionInfo,
    redeliveryHash,
  };
}

/*
 * Helpers
 */

export function dbg<T>(x: T, msg?: string): T {
  if (msg) {
    console.log("[DEBUG] " + msg);
  }
  console.log(x);
  return x;
}
