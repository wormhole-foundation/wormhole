import { BN } from "@project-serum/anchor";
import { ParsedGovernanceVaa, parseGovernanceVaa } from "./governance";
import { ParsedVaa, parseVaa, SignedVaa } from "./wormhole";

export enum TokenBridgePayload {
  Transfer = 1,
  AttestMeta,
  TransferWithPayload,
}

export enum TokenBridgeGovernanceAction {
  RegisterChain = 1,
  UpgradeContract = 2,
}

export interface TokenTransfer {
  payloadType:
    | TokenBridgePayload.Transfer
    | TokenBridgePayload.TransferWithPayload;
  amount: bigint;
  tokenAddress: Buffer;
  tokenChain: number;
  to: Buffer;
  toChain: number;
  fee: bigint | null;
  fromAddress: Buffer | null;
  tokenTransferPayload: Buffer;
}

export function parseTokenTransferPayload(payload: Buffer): TokenTransfer {
  const payloadType = payload.readUInt8(0);
  if (
    payloadType != TokenBridgePayload.Transfer &&
    payloadType != TokenBridgePayload.TransferWithPayload
  ) {
    throw new Error("not token bridge transfer VAA");
  }
  const amount = BigInt(new BN(payload.subarray(1, 33)).toString());
  const tokenAddress = payload.subarray(33, 65);
  const tokenChain = payload.readUInt16BE(65);
  const to = payload.subarray(67, 99);
  const toChain = payload.readUInt16BE(99);
  const fee =
    payloadType == 1
      ? BigInt(new BN(payload.subarray(101, 133)).toString())
      : null;
  const fromAddress = payloadType == 3 ? payload.subarray(101, 133) : null;
  const tokenTransferPayload = payload.subarray(133);
  return {
    payloadType,
    amount,
    tokenAddress,
    tokenChain,
    to,
    toChain,
    fee,
    fromAddress,
    tokenTransferPayload,
  };
}

export interface ParsedTokenTransferVaa extends ParsedVaa, TokenTransfer {}

export function parseTokenTransferVaa(vaa: SignedVaa): ParsedTokenTransferVaa {
  const parsed = parseVaa(vaa);
  return {
    ...parsed,
    ...parseTokenTransferPayload(parsed.payload),
  };
}

export interface AssetMeta {
  payloadType: TokenBridgePayload.AttestMeta;
  tokenAddress: Buffer;
  tokenChain: number;
  decimals: number;
  symbol: string;
  name: string;
}

export function parseAttestMetaPayload(payload: Buffer): AssetMeta {
  const payloadType = payload.readUInt8(0);
  if (payloadType != TokenBridgePayload.AttestMeta) {
    throw new Error("not token bridge attest meta VAA");
  }
  const tokenAddress = payload.subarray(1, 33);
  const tokenChain = payload.readUInt16BE(33);
  const decimals = payload.readUInt8(35);
  const symbol = payload.subarray(36, 68).toString().replace(/\0/g, "");
  const name = payload.subarray(68, 100).toString().replace(/\0/g, "");
  return {
    payloadType,
    tokenAddress,
    tokenChain,
    decimals,
    symbol,
    name,
  };
}

export interface ParsedAssetMetaVaa extends ParsedVaa, AssetMeta {}
export type ParsedAttestMetaVaa = ParsedAssetMetaVaa;

export function parseAttestMetaVaa(vaa: SignedVaa): ParsedAssetMetaVaa {
  const parsed = parseVaa(vaa);
  return {
    ...parsed,
    ...parseAttestMetaPayload(parsed.payload),
  };
}

export interface TokenBridgeRegisterChain {
  foreignChain: number;
  foreignAddress: Buffer;
}

export function parseTokenBridgeRegisterChainGovernancePayload(
  payload: Buffer
): TokenBridgeRegisterChain {
  const foreignChain = payload.readUInt16BE(0);
  const foreignAddress = payload.subarray(2, 34);
  return {
    foreignChain,
    foreignAddress,
  };
}

export interface ParsedTokenBridgeRegisterChainVaa
  extends ParsedGovernanceVaa,
    TokenBridgeRegisterChain {}

export function parseTokenBridgeRegisterChainVaa(
  vaa: SignedVaa
): ParsedTokenBridgeRegisterChainVaa {
  const parsed = parseGovernanceVaa(vaa);
  if (parsed.action != TokenBridgeGovernanceAction.RegisterChain) {
    throw new Error(
      "parsed.action != TokenBridgeGovernanceAction.RegisterChain"
    );
  }
  return {
    ...parsed,
    ...parseTokenBridgeRegisterChainGovernancePayload(parsed.orderPayload),
  };
}

export interface TokenBridgeUpgradeContract {
  newContract: Buffer;
}

export function parseTokenBridgeUpgradeContractGovernancePayload(
  payload: Buffer
): TokenBridgeUpgradeContract {
  const newContract = payload.subarray(0, 32);
  return {
    newContract,
  };
}

export interface ParsedTokenBridgeUpgradeContractVaa
  extends ParsedGovernanceVaa,
    TokenBridgeUpgradeContract {}

export function parseTokenBridgeUpgradeContractVaa(
  vaa: SignedVaa
): ParsedTokenBridgeUpgradeContractVaa {
  const parsed = parseGovernanceVaa(vaa);
  if (parsed.action != TokenBridgeGovernanceAction.UpgradeContract) {
    throw new Error(
      "parsed.action != TokenBridgeGovernanceAction.UpgradeContract"
    );
  }
  return {
    ...parsed,
    ...parseTokenBridgeUpgradeContractGovernancePayload(parsed.orderPayload),
  };
}
