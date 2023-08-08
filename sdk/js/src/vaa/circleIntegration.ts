import { BN } from "@project-serum/anchor";

export enum CircleIntegrationPayload {
  DepositWithPayload = 1,
}

export interface CircleIntegrationDeposit {
  payloadType: CircleIntegrationPayload.DepositWithPayload;
  tokenAddress: Buffer;
  amount: bigint;
  sourceDomain: number;
  targetDomain: number;
  nonce: bigint;
  fromAddress: Buffer;
  mintRecipient: Buffer;
  payloadLen: number;
  depositPayload: Buffer;
}

export function parseCircleIntegrationDepositWithPayload(
  payload: Buffer
): CircleIntegrationDeposit {
  const payloadType = payload.readUInt8(0);
  if (payloadType != CircleIntegrationPayload.DepositWithPayload) {
    throw new Error("not circle integration payload VAA");
  }
  const tokenAddress = payload.subarray(1, 33);
  const amount = BigInt(new BN(payload.subarray(33, 65)).toString());
  const sourceDomain = payload.readUInt32BE(65);
  const targetDomain = payload.readUInt32BE(69);
  const nonce = payload.readBigUInt64BE(73);
  const fromAddress = payload.subarray(81, 113);
  const mintRecipient = payload.subarray(113, 145);
  const payloadLen = payload.readUInt16BE(145);
  const depositPayload = payload.subarray(147);
  return {
    payloadType,
    tokenAddress,
    amount,
    sourceDomain,
    targetDomain,
    nonce,
    fromAddress,
    mintRecipient,
    payloadLen,
    depositPayload,
  };
}
