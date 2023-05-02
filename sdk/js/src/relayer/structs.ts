import { BigNumber, ethers } from "ethers";
import { arrayify } from "ethers/lib/utils";

export enum RelayerPayloadId {
  Delivery = 1,
  Redelivery = 2,
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
  CrossChainRefundSentMaximumBudget,
  CrossChainRefundFailProviderNotSupported,
  CrossChainRefundFailNotEnough
}

export interface VaaKey {
  payloadType: VaaKeyType;
  chainId?: number;
  emitterAddress?: Buffer;
  sequence?: BigNumber;
  vaaHash?: Buffer;
}

export interface DeliveryInstruction {
  targetChain: number;
  targetAddress: Buffer;
  refundAddress: Buffer;
  refundChain: number;
  maximumRefundTarget: BigNumber;
  receiverValueTarget: BigNumber;
  sourceRelayProvider: Buffer;
  targetRelayProvider: Buffer;
  senderAddress: Buffer;
  vaaKeys: VaaKey[];
  consistencyLevel: number;
  executionParameters: ExecutionParameters;
  payload: Buffer;
}

export interface RedeliveryInstruction {
  vaaKey: VaaKey;
  newMaximumRefundTarget: BigNumber;
  newReceiverValueTarget: BigNumber;
  sourceRelayProvider: Buffer;
  targetChain: number;
  executionParameters: ExecutionParameters;
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

export interface ExecutionParameters {
  version: number;
  gasLimit: number;
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
    payloadType: VaaKeyType.EMITTER_SEQUENCE,
    chainId,
    emitterAddress,
    sequence: ethers.BigNumber.from(sequence),
  };
}

export function createVaaKeyFromVaaHash(vaaHash: Buffer): VaaKey {
  return {
    payloadType: VaaKeyType.VAAHASH,
    vaaHash,
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

  const targetChain = bytes.readUInt16BE(idx);
  idx += 2;
  const targetAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const refundChain = bytes.readUInt16BE(idx);
  idx += 2;
  const refundAddress = bytes.slice(idx, idx + 32);
  idx += 32;
  const maximumRefundTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;
  const receiverValueTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;
  const sourceRelayProvider = bytes.slice(idx, idx + 32);
  idx += 32;
  const targetRelayProvider = bytes.slice(idx, idx + 32);
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

  const consistencyLevel = bytes.readUInt8(idx);
  idx += 1;

  let res = parseWormholeRelayerExecutionParameters(bytes, idx);
  const executionParameters = res[0];
  idx = res[1];
  let payload: Buffer;
  [payload, idx] = parsePayload(bytes, idx);

  return {
    targetChain,
    targetAddress,
    refundAddress,
    refundChain,
    maximumRefundTarget,
    receiverValueTarget,
    sourceRelayProvider,
    targetRelayProvider,
    senderAddress,
    vaaKeys: messages,
    executionParameters,
    consistencyLevel,
    payload,
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

  const payloadType = bytes.readUInt8(idx) as VaaKeyType;
  idx += 1;

  dbg(payloadType, "payloadType");
  if (payloadType == VaaKeyType.EMITTER_SEQUENCE) {
    dbg(null, "parsingEmitterSequence");
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
        payloadType,
        chainId,
        emitterAddress,
        sequence,
      },
      idx,
    ];
  } else if (payloadType == VaaKeyType.VAAHASH) {
    const vaaHash = bytes.slice(idx, idx + 32);
    idx += 32;
    return [
      {
        payloadType,
        vaaHash,
      },
      idx,
    ];
  } else {
    throw new Error("Unexpected VaaKey payload type");
  }
}

function parseWormholeRelayerExecutionParameters(
  bytes: Buffer,
  idx: number
): [ExecutionParameters, number] {
  const version = bytes.readUInt8(idx);
  idx += 1;
  const gasLimit = bytes.readUInt32BE(idx);
  idx += 4;
  return [{ version, gasLimit }, idx];
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
  const vaaKey = parsedKey[0];
  idx = parsedKey[1];

  const newMaximumRefundTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  const newReceiverValueTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  const sourceRelayProvider = bytes.slice(idx, idx + 32);
  idx += 32;

  const targetChain: number = bytes.readUInt16BE(idx);
  idx += 2;

  let parsedExecutionParams = parseWormholeRelayerExecutionParameters(
    bytes,
    idx
  );
  const executionParameters = parsedExecutionParams[0];
  idx = parsedExecutionParams[1];

  return {
    vaaKey,
    newMaximumRefundTarget,
    newReceiverValueTarget,
    sourceRelayProvider,
    targetChain,
    executionParameters,
  };
}

export function deliveryInstructionsPrintable(
  ix: DeliveryInstruction
): DeliveryInstructionPrintable {
  return {
    targetChain: ix.targetChain.toString(),
    targetAddress: ix.targetAddress.toString("hex"),
    refundChain: ix.refundChain.toString(),
    refundAddress: ix.refundAddress.toString("hex"),
    maximumRefundTarget: ix.maximumRefundTarget.toString(),
    receiverValueTarget: ix.receiverValueTarget.toString(),
    sourceRelayProvider: ix.sourceRelayProvider.toString("hex"),
    targetRelayProvider: ix.targetRelayProvider.toString("hex"),
    senderAddress: ix.senderAddress.toString("hex"),
    vaaKeys: ix.vaaKeys.map(vaaKeyPrintable),
    consistencyLevel: ix.consistencyLevel.toString(),
    executionParameters: {
      gasLimit: ix.executionParameters.gasLimit.toString(),
      version: ix.executionParameters.version.toString(),
    },
    payload: ix.payload.toString("base64"),
  };
}

export function vaaKeyPrintable(ix: VaaKey): StringLeaves<VaaKey> {
  switch (ix.payloadType) {
    case VaaKeyType.EMITTER_SEQUENCE:
      return {
        payloadType: "EMITTER_SEQUENCE",
        chainId: ix.chainId?.toString(),
        emitterAddress: ix.emitterAddress?.toString("hex"),
        sequence: ix.sequence?.toString(),
      };
    case VaaKeyType.VAAHASH:
      return {
        payloadType: "VAAHASH",
        vaaHash: ix.vaaHash?.toString("hex"),
      };
  }
}

export function redeliveryInstructionPrintable(
  ix: RedeliveryInstruction
): RedeliveryInstructionPrintable {
  return {
    vaaKey: vaaKeyPrintable(ix.vaaKey),
    newMaximumRefundTarget: ix.newMaximumRefundTarget.toString(),
    newReceiverValueTarget: ix.newReceiverValueTarget.toString(),
    sourceRelayProvider: ix.sourceRelayProvider.toString("hex"),
    targetChain: ix.targetChain.toString(),
    executionParameters: {
      gasLimit: ix.executionParameters.gasLimit.toString(),
      version: ix.executionParameters.version.toString(),
    },
  };
}

export type DeliveryOverrideArgs = {
  gasLimit: number;
  newMaximumRefundTarget: BigNumber;
  newReceiverValueTarget: BigNumber;
  redeliveryHash: Buffer;
};

export function packOverrides(overrides: DeliveryOverrideArgs): string {
  const packed = [
    ethers.utils.solidityPack(["uint8"], [1]).substring(2), //version
    ethers.utils.solidityPack(["uint32"], [overrides.gasLimit]).substring(2),
    ethers.utils
      .solidityPack(["uint256"], [overrides.newMaximumRefundTarget])
      .substring(2),
    ethers.utils
      .solidityPack(["uint256"], [overrides.newReceiverValueTarget])
      .substring(2),
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

  const redeliveryHash = bytes.slice(idx, idx + 32);
  idx += 32;

  const newMaximumRefundTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  const newReceiverValueTarget = ethers.BigNumber.from(
    Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
  );
  idx += 32;

  const gasLimit = bytes.readUInt32BE(idx);
  idx += 4;

  return {
    gasLimit,
    newMaximumRefundTarget,
    newReceiverValueTarget,
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
