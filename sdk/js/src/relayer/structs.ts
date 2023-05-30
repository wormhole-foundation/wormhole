import { BigNumber, ethers } from "ethers";
import { arrayify } from "ethers/lib/utils";

export enum RelayerPayloadId {
  Delivery = 1,
  Redelivery = 2,
}

export enum ExecutionInfoVersion {
  EVM_V1 = 0
}

export enum DeliveryStatus {
  WaitingForVAA = "Waiting for VAA",
  PendingDelivery = "Pending Delivery",
  DeliverySuccess = "Delivery Success",
  ReceiverFailure = "Receiver Failure",
  ForwardRequestSuccess = "Forward Request Success",
  ForwardRequestFailure = "Forward Request Failure",
  ThisShouldNeverHappen = "This should never happen. Contact Support.",
  DeliveryDidntHappenWithinRange = "Delivery didn't happen within given block range",
}

export enum RefundStatus {
  RefundSent,
  RefundFail,
  CrossChainRefundSent,
  CrossChainRefundFailProviderNotSupported,
  CrossChainRefundFailNotEnough
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
  vaaKeys: VaaKey[];
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
  gasLimit: number;
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

  let messages = [] as VaaKey[];
  for (let i = 0; i < numMessages; ++i) {
    const res = parseVaaKey(bytes, idx);
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
    vaaKeys: messages
  };
}

function parsePayload(bytes: Buffer, idx: number): [Buffer, number] {
  const length = bytes.readUInt32BE(idx);
  idx += 4;
  const payload = bytes.slice(idx, idx + length);
  idx += length;
  return [payload, idx];
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

export function parseEVMExecutionInfoV1(
  bytes: Buffer,
  idx: number
): [EVMExecutionInfoV1, number] {
  idx += 31;
  const version = bytes.readUInt8(idx);
  idx += 1;
  if(version !== ExecutionInfoVersion.EVM_V1) {
    throw new Error("Unexpected Execution Info version");
  }
  idx += 28;
  const gasLimit = bytes.readUInt32BE(idx);
  idx += 4;
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
    newSenderAddress
  };
}

export function executionInfoToString(
  encodedExecutionInfo: Buffer
): string {
  const [parsed,] = parseEVMExecutionInfoV1(encodedExecutionInfo, 0)
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
    vaaKeys: ix.vaaKeys.map(vaaKeyPrintable),
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
    newSenderAddress: ix.newSenderAddress.toString("hex")
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

export function parseForwardFailureError(
  bytes: Buffer
): string {
  let idx = 4;
  console.log(bytes.length);
  const amountOfFunds = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;
  const amountOfFundsNeeded = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  return `Not enough funds leftover for forward: Had ${amountOfFunds.toString()} wei and needed ${amountOfFundsNeeded.toString()} wei.`
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
    redeliveryHash
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
