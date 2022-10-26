import { BN } from "@project-serum/anchor";
import { ParsedGovernanceVaa, parseGovernanceVaa } from "./governance";
import {
  parseTokenBridgeRegisterChainGovernancePayload,
  parseTokenBridgeUpgradeContractGovernancePayload,
  TokenBridgeRegisterChain,
  TokenBridgeUpgradeContract,
} from "./tokenBridge";
import { ParsedVaa, parseVaa, SignedVaa } from "./wormhole";

export enum NftBridgePayload {
  Transfer = 1,
}

export enum NftBridgeGovernanceAction {
  RegisterChain = 1,
  UpgradeContract = 2,
}

export interface NftTransfer {
  payloadType: NftBridgePayload.Transfer;
  tokenAddress: Buffer;
  tokenChain: number;
  symbol: string;
  name: string;
  tokenId: bigint;
  uri: string;
  to: Buffer;
  toChain: number;
}

export function parseNftTransferPayload(payload: Buffer): NftTransfer {
  const payloadType = payload.readUInt8(0);
  if (payloadType != NftBridgePayload.Transfer) {
    throw new Error("not nft bridge transfer VAA");
  }
  const tokenAddress = payload.subarray(1, 33);
  const tokenChain = payload.readUInt16BE(33);
  const symbol = payload.subarray(35, 67).toString().replace(/\0/g, "");
  const name = payload.subarray(67, 99).toString().replace(/\0/g, "");
  const tokenId = BigInt(new BN(payload.subarray(99, 131)).toString());
  const uriLen = payload.readUInt8(131);
  const uri = payload.subarray(132, 132 + uriLen).toString();
  const uriEnd = 132 + uriLen;
  const to = payload.subarray(uriEnd, uriEnd + 32);
  const toChain = payload.readUInt16BE(uriEnd + 32);
  return {
    payloadType,
    tokenAddress,
    tokenChain,
    name,
    symbol,
    tokenId,
    uri,
    to,
    toChain,
  };
}

export interface ParsedNftTransferVaa extends ParsedVaa, NftTransfer {}

export function parseNftTransferVaa(vaa: SignedVaa): ParsedNftTransferVaa {
  const parsed = parseVaa(vaa);
  return {
    ...parsed,
    ...parseNftTransferPayload(parsed.payload),
  };
}

export interface NftRegisterChain extends TokenBridgeRegisterChain {}
export interface ParsedNftBridgeRegisterChainVaa
  extends ParsedGovernanceVaa,
    NftRegisterChain {}

export function parseNftBridgeRegisterChainGovernancePayload(
  payload: Buffer
): NftRegisterChain {
  return parseTokenBridgeRegisterChainGovernancePayload(payload);
}

export function parseNftBridgeRegisterChainVaa(
  vaa: SignedVaa
): ParsedNftBridgeRegisterChainVaa {
  const parsed = parseGovernanceVaa(vaa);
  if (parsed.action != NftBridgeGovernanceAction.RegisterChain) {
    throw new Error("parsed.action != NftBridgeGovernanceAction.RegisterChain");
  }
  return {
    ...parsed,
    ...parseNftBridgeRegisterChainGovernancePayload(parsed.orderPayload),
  };
}

export interface NftBridgeUpgradeContract extends TokenBridgeUpgradeContract {}

export function parseNftBridgeUpgradeContractGovernancePayload(
  payload: Buffer
): NftBridgeUpgradeContract {
  return parseTokenBridgeUpgradeContractGovernancePayload(payload);
}

export interface ParsedNftBridgeUpgradeContractVaa
  extends ParsedGovernanceVaa,
    NftBridgeUpgradeContract {}

export function parseNftBridgeUpgradeContractVaa(
  vaa: SignedVaa
): ParsedNftBridgeUpgradeContractVaa {
  const parsed = parseGovernanceVaa(vaa);
  if (parsed.action != NftBridgeGovernanceAction.UpgradeContract) {
    throw new Error(
      "parsed.action != NftBridgeGovernanceAction.UpgradeContract"
    );
  }
  return {
    ...parsed,
    ...parseNftBridgeUpgradeContractGovernancePayload(parsed.orderPayload),
  };
}
