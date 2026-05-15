// VAA payload encoding/decoding utilities

// Address types for output
export const ADDRESS_TYPE_P2PKH = 0;
export const ADDRESS_TYPE_P2SH = 1;

export interface UnlockPayloadInput {
  originalRecipientAddress: string; // 32 bytes hex
  transactionId: string; // 32 bytes hex
  vout: number;
}

export interface UnlockPayloadOutput {
  amount: bigint;
  addressType: number;
  address: string; // hex
}

export interface UnlockPayload {
  destinationChain: number;
  delegatedManagerSet: number;
  inputs: UnlockPayloadInput[];
  outputs: UnlockPayloadOutput[];
}

export interface ParsedVAA {
  version: number;
  guardianSetIndex: number;
  signatures: { guardianIndex: number; signature: Buffer }[];
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: string;
  sequence: bigint;
  consistencyLevel: number;
  payload: Buffer;
}

// Encode the VAA payload for Dogecoin unlock
export function encodeUnlockPayload(params: UnlockPayload): Buffer {
  const parts: Buffer[] = [];

  // Prefix "UTX0" (4 bytes)
  parts.push(Buffer.from("UTX0", "ascii"));

  // destination_chain (uint16 BE)
  const destChainBuf = Buffer.alloc(2);
  destChainBuf.writeUInt16BE(params.destinationChain);
  parts.push(destChainBuf);

  // delegated_manager_set (uint32 BE)
  const managerSetBuf = Buffer.alloc(4);
  managerSetBuf.writeUInt32BE(params.delegatedManagerSet);
  parts.push(managerSetBuf);

  // len_input (uint32 BE)
  const lenInputBuf = Buffer.alloc(4);
  lenInputBuf.writeUInt32BE(params.inputs.length);
  parts.push(lenInputBuf);

  // inputs
  for (const input of params.inputs) {
    // original_recipient_address (32 bytes)
    parts.push(Buffer.from(input.originalRecipientAddress, "hex"));
    // transaction_id (32 bytes)
    parts.push(Buffer.from(input.transactionId, "hex"));
    // vout (uint32 BE)
    const voutBuf = Buffer.alloc(4);
    voutBuf.writeUInt32BE(input.vout);
    parts.push(voutBuf);
  }

  // len_output (uint32 BE)
  const lenOutputBuf = Buffer.alloc(4);
  lenOutputBuf.writeUInt32BE(params.outputs.length);
  parts.push(lenOutputBuf);

  // outputs
  for (const output of params.outputs) {
    // amount (uint64 BE)
    const amountBuf = Buffer.alloc(8);
    amountBuf.writeBigUInt64BE(output.amount);
    parts.push(amountBuf);
    // address_type (uint32 BE)
    const addrTypeBuf = Buffer.alloc(4);
    addrTypeBuf.writeUInt32BE(output.addressType);
    parts.push(addrTypeBuf);
    // address (length determined by address_type: 20 for P2PKH/P2SH, 32 for P2WSH/P2TR)
    const addrBuf = Buffer.from(output.address, "hex");
    parts.push(addrBuf);
  }

  return Buffer.concat(parts.map((p) => new Uint8Array(p)));
}

// Decode VAA payload
export function decodeUnlockPayload(payload: Buffer): UnlockPayload {
  let offset = 0;

  // Prefix (4 bytes)
  const prefix = payload.subarray(offset, offset + 4).toString("ascii");
  if (prefix !== "UTX0") {
    throw new Error(`Invalid payload prefix: ${prefix}`);
  }
  offset += 4;

  // destination_chain (uint16 BE)
  const destinationChain = payload.readUInt16BE(offset);
  offset += 2;

  // delegated_manager_set (uint32 BE)
  const delegatedManagerSet = payload.readUInt32BE(offset);
  offset += 4;

  // len_input (uint32 BE)
  const lenInput = payload.readUInt32BE(offset);
  offset += 4;

  // inputs
  const inputs: UnlockPayloadInput[] = [];
  for (let i = 0; i < lenInput; i++) {
    const originalRecipientAddress = payload
      .subarray(offset, offset + 32)
      .toString("hex");
    offset += 32;
    const transactionId = payload.subarray(offset, offset + 32).toString("hex");
    offset += 32;
    const vout = payload.readUInt32BE(offset);
    offset += 4;
    inputs.push({ originalRecipientAddress, transactionId, vout });
  }

  // len_output (uint32 BE)
  const lenOutput = payload.readUInt32BE(offset);
  offset += 4;

  // outputs
  const outputs: UnlockPayloadOutput[] = [];
  for (let i = 0; i < lenOutput; i++) {
    const amount = payload.readBigUInt64BE(offset);
    offset += 8;
    const addressType = payload.readUInt32BE(offset);
    offset += 4;
    // Address length determined by address_type: 20 for P2PKH/P2SH, 32 for P2WSH/P2TR
    const addrLen = addressType <= 1 ? 20 : 32;
    const address = payload.subarray(offset, offset + addrLen).toString("hex");
    offset += addrLen;
    outputs.push({ amount, addressType, address });
  }

  return { destinationChain, delegatedManagerSet, inputs, outputs };
}

// Parse VAA structure
export function parseVAA(vaaBytes: Buffer): ParsedVAA {
  let offset = 0;

  // Version (1 byte)
  const version = vaaBytes.readUInt8(offset);
  offset += 1;

  // Guardian set index (4 bytes BE)
  const guardianSetIndex = vaaBytes.readUInt32BE(offset);
  offset += 4;

  // Number of signatures (1 byte)
  const numSignatures = vaaBytes.readUInt8(offset);
  offset += 1;

  // Signatures (66 bytes each: 1 byte index + 65 bytes signature)
  const signatures: { guardianIndex: number; signature: Buffer }[] = [];
  for (let i = 0; i < numSignatures; i++) {
    const guardianIndex = vaaBytes.readUInt8(offset);
    offset += 1;
    const signature = vaaBytes.subarray(offset, offset + 65);
    offset += 65;
    signatures.push({ guardianIndex, signature: Buffer.from(signature) });
  }

  // Timestamp (4 bytes BE)
  const timestamp = vaaBytes.readUInt32BE(offset);
  offset += 4;

  // Nonce (4 bytes BE)
  const nonce = vaaBytes.readUInt32BE(offset);
  offset += 4;

  // Emitter chain (2 bytes BE)
  const emitterChain = vaaBytes.readUInt16BE(offset);
  offset += 2;

  // Emitter address (32 bytes)
  const emitterAddress = vaaBytes.subarray(offset, offset + 32).toString("hex");
  offset += 32;

  // Sequence (8 bytes BE)
  const sequence = vaaBytes.readBigUInt64BE(offset);
  offset += 8;

  // Consistency level (1 byte)
  const consistencyLevel = vaaBytes.readUInt8(offset);
  offset += 1;

  // Payload (rest of the data)
  const payload = vaaBytes.subarray(offset);

  return {
    version,
    guardianSetIndex,
    signatures,
    timestamp,
    nonce,
    emitterChain,
    emitterAddress,
    sequence,
    consistencyLevel,
    payload: Buffer.from(payload),
  };
}
