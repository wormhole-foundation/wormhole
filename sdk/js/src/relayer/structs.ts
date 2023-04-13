import { BigNumber, ethers } from "ethers";
import { arrayify } from "ethers/lib/utils";

export enum RelayerPayloadId {
  Delivery = 1,
  // DeliveryStatus = 3,
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

export interface DeliveryInstructionsContainer {
  payloadId: number; // 1
  senderAddress: Buffer;
  sourceProvider: Buffer;
  messages: MessageInfo[];
  instructions: DeliveryInstruction[];
}

export interface MessageInfo {
  payloadType: MessageInfoType;
  chainId ?: number;
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
  targetRelayProvider: Buffer;
  executionParameters: ExecutionParameters;
  payload: Buffer;
}

export interface ExecutionParameters {
  version: number;
  gasLimit: number;
}

export enum MessageInfoType {
  EMITTER_SEQUENCE = 0,
  VAAHASH = 1,
}

export function parseWormholeRelayerPayloadType(
  stringPayload: string | Buffer | Uint8Array
): RelayerPayloadId {
  const payload =
    typeof stringPayload === "string" ? arrayify(stringPayload) : stringPayload;
  if (payload[0] == 0 || payload[0] >= 3) {
    throw new Error("Unrecogned payload type " + payload[0]);
  }
  return payload[0];
}

export function parseWormholeRelayerSend(
  bytes: Buffer
): DeliveryInstructionsContainer {
  let idx = 0;
  const payloadId = bytes.readUInt8(idx);
  if (payloadId !== RelayerPayloadId.Delivery) {
    throw new Error(
      `Expected Delivery payload type (${RelayerPayloadId.Delivery}), found: ${payloadId}`
    );
  }
  idx += 1;

  const senderAddress = bytes.slice(idx, idx + 32);
  idx += 32;

  const sourceProvider = bytes.slice(idx, idx + 32);
  idx += 32;

  const numMessages = bytes.readUInt8(idx);
  idx += 1;

  const numInstructions = bytes.readUInt8(idx);
  idx += 1;

  let messages = [] as MessageInfo[];
  for (let i = 0; i < numMessages; ++i) {
    const res = parseMessageInfo(bytes, idx);
    idx = res[1];
    messages.push(res[0]);
  }

  let instructions = [] as DeliveryInstruction[];
  for (let i = 0; i < numInstructions; ++i) {
    const targetChain = bytes.readUInt16BE(idx);
    idx += 2;
    const targetAddress = bytes.slice(idx, idx + 32);
    idx += 32;
    const refundAddress = bytes.slice(idx, idx + 32);
    idx += 32;
    const refundChain = bytes.readUInt16BE(idx);
    idx += 2;
    const maximumRefundTarget = ethers.BigNumber.from(
      Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
    );
    idx += 32;
    const receiverValueTarget = ethers.BigNumber.from(
      Uint8Array.prototype.subarray.call(bytes, idx, idx + 32)
    );
    idx += 32;
    const targetRelayProvider = bytes.slice(idx, idx + 32);
    idx += 32;
    let res = parseWormholeRelayerExecutionParameters(bytes, idx);
    const executionParameters = res[0];
    idx = res[1];
    let payload: Buffer;
    [payload, idx] = parsePayload(bytes, idx);
    instructions.push(
      // dumb typechain format
      {
        targetChain,
        targetAddress,
        refundAddress,
        refundChain,
        maximumRefundTarget,
        receiverValueTarget,
        targetRelayProvider,
        executionParameters,
        payload,
      }
    );
  }

  return {
    payloadId,
    senderAddress,
    sourceProvider,
    messages,
    instructions,
  };
}

function parsePayload(bytes: Buffer, idx: number): [Buffer, number] {
  const length = bytes.readUInt32BE(idx);
  idx += 4;
  const payload = bytes.slice(idx, idx + length);
  idx += length;
  return [payload, idx];
}

function parseMessageInfo(bytes: Buffer, idx: number): [MessageInfo, number] {
  const version = bytes.readUInt8(idx);
  idx += 1;

  const payloadType = bytes.readUInt8(idx) as MessageInfoType;
  idx += 1;

  dbg(payloadType, "payloadType");
  if (payloadType == MessageInfoType.EMITTER_SEQUENCE) {
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
  } else if (payloadType == MessageInfoType.VAAHASH) {
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
    throw new Error("Unexpected MessageInfo payload type");
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
