import { ParsedVaa, parseVaa, SignedVaa } from "./wormhole";

export interface Governance {
  module: string;
  action: number;
  chain: number;
  orderPayload: Buffer;
}

export interface ParsedGovernanceVaa extends ParsedVaa, Governance {}

export function parseGovernanceVaa(vaa: SignedVaa): ParsedGovernanceVaa {
  const parsed = parseVaa(vaa);
  return {
    ...parsed,
    ...parseGovernancePayload(parsed.payload),
  };
}

export function parseGovernancePayload(payload: Buffer): Governance {
  const module = payload.subarray(0, 32).toString().replace(/\0/g, "");
  const action = payload.readUInt8(32);
  const chain = payload.readUInt16BE(33);
  const orderPayload = payload.subarray(35);
  return {
    module,
    action,
    chain,
    orderPayload,
  };
}
