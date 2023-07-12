import { BigNumber } from "@ethersproject/bignumber";
import {
  NftTransfer,
  TokenTransfer,
  parseNftTransferPayload,
  parseTokenTransferPayload,
} from "../vaa";

export const METADATA_REPLACE = new RegExp("\u0000", "g");

/**
 * NFTTransferPayload is the payload data for a NFT transfer VAA.
 */
export type NFTTransferPayload = Pick<
  NftTransfer,
  "symbol" | "name" | "uri"
> & {
  originAddress: string;
  originChain: number;
  targetAddress: string;
  fee?: BigNumber;
  targetChain: number;
  fromAddress?: string;
  tokenId: BigNumber;
};

// note: actual first byte is message type
//     0   [u8; 32] token_address
//     32  u16      token_chain
//     34  [u8; 32] symbol
//     66  [u8; 32] name
//     98  u256     tokenId
//     130 u8       uri_len
//     131 [u8;len] uri
//     ?   [u8; 32] recipient
//     ?   u16      recipient_chain
export function parseNFTPayload(payload: Buffer): NFTTransferPayload {
  const parsed = parseNftTransferPayload(payload);
  return {
    originAddress: parsed.tokenAddress.toString("hex"),
    originChain: parsed.tokenChain,
    symbol: parsed.symbol,
    name: parsed.name,
    tokenId: BigNumber.from(parsed.tokenId),
    uri: parsed.uri,
    targetAddress: parsed.to.toString("hex"),
    targetChain: parsed.toChain,
  };
}

/**
 * TokenTransferPayload is the payload data for a Token transfer VAA.
 */
export type TokenTransferPayload = Pick<TokenTransfer, "amount"> & {
  originAddress: string;
  originChain: number;
  targetAddress: string;
  targetChain: number;
  fromAddress?: string;
  fee?: BigInt;
};

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee
export function parseTransferPayload(payload: Buffer): TokenTransferPayload {
  const parsed = parseTokenTransferPayload(payload);
  return {
    amount: parsed.amount,
    originAddress: parsed.tokenAddress.toString("hex"),
    originChain: parsed.tokenChain,
    targetAddress: parsed.to.toString("hex"),
    targetChain: parsed.toChain,
    fee: parsed.fee === null ? undefined : parsed.fee,
    fromAddress:
      parsed.fromAddress === null
        ? undefined
        : parsed.fromAddress.toString("hex"),
  };
}
