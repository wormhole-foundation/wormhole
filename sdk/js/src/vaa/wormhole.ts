import { keccak256 } from "../utils";

export { isBytes } from "ethers/lib/utils";

export interface GuardianSignature {
  index: number;
  signature: Buffer;
}

interface ParsedVaaHeader {
  version: number;
  guardianSetIndex: number;
  guardianSignatures: GuardianSignature[];
}

export type ParsedVaaV1 = ParsedVaaHeader & ParsedVaaV3;

// BatchVAA
export interface ParsedVaaV2 {
  version: number;
  guardianSetIndex: number;
  guardianSignatures: GuardianSignature[];
  hashes: Buffer[];
  observations: ParsedVaaV3[];
}

// Headless VAA inside Batch
export interface ParsedVaaV3 {
  version: number;
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: Buffer;
  sequence: bigint;
  consistencyLevel: number;
  payload: Buffer;
  hash: Buffer;
}

type ParsedVaa = ParsedVaaV1 | ParsedVaaV2 | ParsedVaaV3;

export type SignedVaa = Uint8Array | Buffer;

export function parseVaa(vaa: SignedVaa): ParsedVaa {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const version = signedVaa[0];
  switch (version) {
    case 1:
      // Traditional VAA
      return parseVaaV1(signedVaa);
    case 2:
      // Batch VAA
      return parseVaaV2(signedVaa);
    default:
      throw new Error("Unrecognized Vaa Version");
  }
}

export function parseVaaV1(vaa: SignedVaa): ParsedVaaV1 {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const {
    header: { version, guardianSetIndex, guardianSignatures },
    headerSize,
  } = parseHeader(signedVaa);
  if (version !== 1) {
    throw new Error("trying to parse a different version vaa");
  }

  const body = signedVaa.subarray(headerSize);

  return {
    version,
    guardianSetIndex,
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

const minBatchVAALength = 94; // HEADER + BATCH

// Batch VAA Parsing. Whitepaper: https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0008_batch_messaging.md#payloads-encoded-messages
export function parseVaaV2(vaa: SignedVaa): ParsedVaaV2 {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  if (signedVaa.length < minBatchVAALength) {
    throw new Error("BatchVAA.Observation is too short");
  }

  const {
    header: { version, guardianSetIndex, guardianSignatures },
    headerSize,
  } = parseHeader(signedVaa);
  if (version !== 2) {
    throw new Error("trying to parse a different version vaa");
  }

  // calculate when hashes and observations begin and end
  const lenHashes = signedVaa[headerSize];
  const hashLength = 32;
  const hashesStartAt = headerSize + 1;
  const observationsStartAt = hashesStartAt + lenHashes * hashLength;

  // parse hashes
  const hashesBytes = signedVaa.subarray(hashesStartAt, observationsStartAt);
  const hashes = [];
  for (let i = 0; i < lenHashes; ++i) {
    const start = i * hashLength;
    hashes.push(hashesBytes.subarray(start, start + hashLength));
  }

  // parse observations
  const observationsBytes = signedVaa.subarray(observationsStartAt);
  const lenObservations = observationsBytes[0];
  if (lenObservations !== lenHashes) {
    throw new Error(
      "failed unmarshaling BatchVAA, observations differs from hashes"
    );
  }
  const observations = [];
  let offset = 1;
  for (let i = 0; i < lenObservations; i++) {
    const index = observationsBytes.readUInt8(offset);
    offset += 1;
    const size = observationsBytes.readUInt32BE(offset);
    offset += 4;
    // TODO checks & validation
    const vaaBodyBytes = observationsBytes.subarray(offset, offset + size);
    const observation = parseVaaV3(vaaBodyBytes);
    observations[index] = observation;
    offset += size;
  }

  return {
    version,
    guardianSetIndex,
    guardianSignatures,
    hashes,
    observations,
  };
}

function parseHeader(signedVaa: Buffer): {
  header: ParsedVaaHeader;
  headerSize: number;
} {
  const signaturesStartAt = 6;
  const version = signedVaa[0];
  if (version !== 2) {
    throw new Error("trying to parse a different version vaa");
  }
  const guardianSetIndex = signedVaa.readUInt32BE(1);
  // parse signatures
  const lenSigners = signedVaa[5];
  const sigLength = 66;
  const guardianSignatures = [];
  for (let i = 0; i < lenSigners; ++i) {
    const start = signaturesStartAt + i * sigLength;
    guardianSignatures.push({
      index: signedVaa[start],
      signature: signedVaa.subarray(start + 1, start + sigLength),
    });
  }
  return {
    header: {
      version,
      guardianSetIndex,
      guardianSignatures,
    },
    headerSize: signaturesStartAt + sigLength * lenSigners,
  };
}

function parseVaaV3(vaaBodyBytes: any): ParsedVaaV3 {
  return {
    version: 3,
    timestamp: vaaBodyBytes.readUInt32BE(0),
    nonce: vaaBodyBytes.readUInt32BE(4),
    emitterChain: vaaBodyBytes.readUInt16BE(8),
    emitterAddress: vaaBodyBytes.subarray(10, 42),
    sequence: vaaBodyBytes.readBigUInt64BE(42),
    consistencyLevel: vaaBodyBytes[50],
    payload: vaaBodyBytes.subarray(51),
    hash: keccak256(vaaBodyBytes),
  };
}
