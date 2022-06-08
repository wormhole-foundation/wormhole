import { ChainId } from "@certusone/wormhole-sdk";
import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";

//TODO move these to the official SDK
export async function parseVaaTyped(signedVAA: Uint8Array) {
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVAA);
  return {
    timestamp: parseInt(parsedVAA.timestamp),
    nonce: parseInt(parsedVAA.nonce),
    emitterChain: parseInt(parsedVAA.emitter_chain) as ChainId,
    emitterAddress: parsedVAA.emitter_address, //This will be in wormhole HEX format
    sequence: parseInt(parsedVAA.sequence),
    consistencyLevel: parseInt(parsedVAA.consistency_level),
    payload: parsedVAA.payload,
  };
}

export type ParsedVaa<T> = {
  timestamp: number;
  nonce: number;
  emitterChain: ChainId;
  emitterAddress: Uint8Array;
  sequence: number;
  consistencyLevel: number;
  payload: T;
};

export type ParsedTransferPayload = {
  amount: BigInt;
  originAddress: string; // hex
  originChain: ChainId;
  targetAddress: string; // hex
  targetChain: ChainId;
  fee?: BigInt;
};

/** Type guard function to ensure an object is of type ParsedTransferPayload */
function IsParsedTransferPayload(
  payload: any
): payload is ParsedTransferPayload {
  return (
    typeof (payload as ParsedTransferPayload).amount == "bigint" &&
    typeof (payload as ParsedTransferPayload).originAddress == "string" &&
    typeof (payload as ParsedTransferPayload).originChain == "number" &&
    typeof (payload as ParsedTransferPayload).targetAddress == "string" &&
    typeof (payload as ParsedTransferPayload).targetChain == "number"
  );
}
