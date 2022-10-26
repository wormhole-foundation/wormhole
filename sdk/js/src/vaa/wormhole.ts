import { keccak256 } from "../utils";

export { isBytes } from "ethers/lib/utils";

export interface GuardianSignature {
  index: number;
  signature: Buffer;
}

export interface ParsedVaa {
  version: number;
  guardianSetIndex: number;
  guardianSignatures: GuardianSignature[];
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: Buffer;
  sequence: bigint;
  consistencyLevel: number;
  payload: Buffer;
  hash: Buffer;
}

export type SignedVaa = Uint8Array | Buffer;

export function parseVaa(vaa: SignedVaa): ParsedVaa {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;

  const guardianSignatures: GuardianSignature[] = [];
  for (let i = 0; i < numSigners; ++i) {
    const start = sigStart + i * sigLength;
    guardianSignatures.push({
      index: signedVaa[start],
      signature: signedVaa.subarray(start + 1, start + 66),
    });
  }

  const body = signedVaa.subarray(sigStart + sigLength * numSigners);

  return {
    version: signedVaa[0],
    guardianSetIndex: signedVaa.readUInt32BE(1),
    guardianSignatures,
    timestamp: body.readUInt32BE(0),
    nonce: body.readUInt32BE(4),
    emitterChain: body.readUInt16BE(8),
    emitterAddress: body.subarray(10, 42),
    sequence: body.readBigUInt64BE(42),
    consistencyLevel: body[50],
    payload: body.subarray(51),
    hash: keccak256(body),
  };
}
