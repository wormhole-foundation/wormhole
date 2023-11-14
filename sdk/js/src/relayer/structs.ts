import { BigNumber, ethers } from "ethers";

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
}

export enum RefundStatus {
  RefundSent = "Refund Sent",
  RefundFail = "Refund Fail",
  CrossChainRefundSent = "Cross Chain Refund Sent",
  CrossChainRefundFailProviderNotSupported = "Cross Chain Refund Fail - Provider does not support the refund chain",
  CrossChainRefundFailNotEnough = "Cross Chain Refund Fail - Refund too low for cross chain refund",
  RefundAddressNotProvided = "No refund address provided",
  InvalidRefundStatus = "Invalid refund status",
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
    : index === 5
    ? RefundStatus.RefundAddressNotProvided
    : RefundStatus.InvalidRefundStatus;
}

export enum KeyType {
  VAA = 1,
  CCTP = 2,
}
export interface MessageKey {
  keyType: KeyType | number;
  key: ethers.BytesLike;
}

export interface VaaKey {
  chainId: number;
  emitterAddress: Buffer;
  sequence: BigNumber;
}

export interface CCTPKey {
  domain: number;
  nonce: ethers.BigNumber;
}

export interface CCTPMessage {
  message: Buffer;
  signature: Buffer;
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
  [Property in keyof Omit<DeliveryInstruction, "messageKeys">]: StringLeaves<
    DeliveryInstruction[Property]
  >;
} & { messageKeys: ReturnType<typeof messageKeyPrintable>[] };

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
    typeof stringPayload === "string"
      ? ethers.utils.arrayify(stringPayload)
      : stringPayload;
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
  let messageKeys = [] as MessageKey[];
  for (let i = 0; i < numMessages; ++i) {
    const res = parseMessageKey(bytes, idx);
    idx = res[1];
    messageKeys.push(res[0]);
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
    messageKeys: messageKeys,
  };
}

function parsePayload(bytes: Buffer, idx: number): [Buffer, number] {
  const length = bytes.readUInt32BE(idx);
  idx += 4;
  const payload = bytes.slice(idx, idx + length);
  idx += length;
  return [payload, idx];
}

export function parseMessageKey(
  _bytes: ethers.utils.BytesLike,
  idx: number
): [MessageKey, number] {
  const bytes = Buffer.from(ethers.utils.arrayify(_bytes));
  const keyType = bytes.readUInt8(idx);
  idx += 1;
  if (keyType === KeyType.VAA) {
    const vaaKeyLength = 2 + 32 + 8;
    return [
      { keyType, key: bytes.slice(idx, idx + vaaKeyLength) },
      idx + 2 + 32 + 8,
    ];
  } else {
    const len = bytes.readUInt32BE(idx);
    idx += 4;
    return [{ keyType, key: bytes.slice(idx, idx + len) }, idx + len];
  }
}

export function packMessageKey(key: MessageKey): string {
  const encodedKey = ethers.utils.arrayify(key.key);
  const bytes = Buffer.alloc(1 + 4 + encodedKey.length);
  let idx = 0;
  bytes.writeUInt8(key.keyType, idx);
  idx += 1;
  if (key.keyType === KeyType.VAA) {
    bytes.fill(encodedKey, idx);
  } else {
    const encodedKey = ethers.utils.arrayify(key.key);
    bytes.writeUInt32BE(encodedKey.length, idx);
    idx += 4;
    bytes.fill(encodedKey, idx);
  }
  return ethers.utils.hexlify(bytes);
}

export function parseCCTPKey(_bytes: ethers.BytesLike): CCTPKey {
  const bytes = Buffer.from(ethers.utils.arrayify(_bytes));
  const domain = bytes.readUInt32BE(0);
  const nonce = ethers.BigNumber.from(bytes.readBigUInt64BE(4));
  return { domain, nonce };
}

export function packCCTPKey(key: CCTPKey): string {
  const buf = Buffer.alloc(4 + 8);
  buf.writeUInt32BE(key.domain, 0);
  buf.writeBigUInt64BE(key.nonce.toBigInt(), 4);
  return ethers.utils.hexlify(buf);
}

export function parseVaaKey(_bytes: ethers.BytesLike): VaaKey {
  const bytes = Buffer.from(ethers.utils.arrayify(_bytes));
  let idx = 0;
  const chainId = bytes.readUInt16BE(idx);
  idx += 2;
  const emitterAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const sequence = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 8)
  );
  idx += 8;
  return {
    chainId,
    emitterAddress,
    sequence,
  };
}

export function packVaaKey(vaaKey: VaaKey): string {
  const bytes = Buffer.alloc(2 + 32 + 8);
  bytes.writeUInt16BE(vaaKey.chainId, 0);
  bytes.fill(vaaKey.emitterAddress, 2, 34);
  bytes.writeBigUInt64BE(vaaKey.sequence.toBigInt(), 34);
  return ethers.utils.hexlify(bytes);
}

export function packCCTPMessage(message: CCTPMessage): string {
  return ethers.utils.defaultAbiCoder.encode(
    ["bytes", "bytes"],
    [message.message, message.signature]
  );
}

export function parseCCTPMessage(bytes: ethers.BytesLike): CCTPMessage {
  const [message, signature] = ethers.utils.defaultAbiCoder.decode(
    ["bytes", "bytes"],
    bytes
  );
  return { message, signature };
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

export function packEVMExecutionInfoV1(info: EVMExecutionInfoV1): string {
  return ethers.utils.defaultAbiCoder.encode(
    ["uint8", "uint256", "uint256"],
    [
      ExecutionInfoVersion.EVM_V1,
      info.gasLimit,
      info.targetChainRefundPerGasUnused,
    ]
  );
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

  let parsedMessageKey: MessageKey;
  [parsedMessageKey, idx] = parseMessageKey(bytes, idx);
  const key: VaaKey = parseVaaKey(parsedMessageKey.key);

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
    extraReceiverValue: ix.extraReceiverValue.toString(),
    encodedExecutionInfo: executionInfoToString(ix.encodedExecutionInfo),
    refundChainId: ix.refundChainId.toString(),
    refundAddress: ix.refundAddress.toString("hex"),
    refundDeliveryProvider: ix.refundDeliveryProvider.toString("hex"),
    sourceDeliveryProvider: ix.sourceDeliveryProvider.toString("hex"),
    senderAddress: ix.senderAddress.toString("hex"),
    messageKeys: ix.messageKeys.map(messageKeyPrintable),
  };
}

export function messageKeyPrintable(
  ix: MessageKey
): StringLeaves<(VaaKey | CCTPKey | { key: string }) & { keyType: number }> {
  switch (ix.keyType) {
    case KeyType.VAA:
      return {
        keyType: "VAA",
        ...(vaaKeyPrintable(parseVaaKey(ix.key)) as {
          [P in keyof VaaKey]: StringLeaves<VaaKey[P]>;
        }),
      };
    case KeyType.CCTP:
      return {
        keyType: "CCTP",
        ...(cctpKeyPrintable(parseCCTPKey(ix.key)) as {
          [P in keyof CCTPKey]: StringLeaves<CCTPKey[P]>;
        }),
      };
    default:
      return {
        keyType: ix.keyType.toString(),
        key: ethers.utils.hexlify(ix.key),
      };
  }
}

export function vaaKeyPrintable(ix: VaaKey): StringLeaves<VaaKey> {
  return {
    chainId: ix.chainId?.toString(),
    emitterAddress: ix.emitterAddress?.toString("hex"),
    sequence: ix.sequence?.toString(),
  };
}

export function cctpKeyPrintable(ix: CCTPKey): StringLeaves<CCTPKey> {
  return {
    domain: ix.domain.toString(),
    nonce: ix.nonce.toString(),
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
