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

export interface VaaKey {
  payloadType: VaaKeyType;
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
  sourceRelayProvider: Buffer;
  targetRelayProvider: Buffer;
  senderAddress: Buffer;
  vaaKeys: VaaKey[];
  consistencyLevel: number;
  executionParameters: ExecutionParameters;
  payload: Buffer;
}

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
  if (payload[0] == 0 || payload[0] >= 3) {
    throw new Error("Unrecogned payload type " + payload[0]);
  }
  return payload[0];
}

export function parseWormholeRelayerSend(
  bytes: Buffer
): DeliveryInstruction {
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
