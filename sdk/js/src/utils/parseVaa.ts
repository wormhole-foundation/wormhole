import { BigNumber } from "@ethersproject/bignumber";
import { parseNftTransferPayload, parseTokenTransferPayload } from "../vaa";

export const METADATA_REPLACE = new RegExp("\u0000", "g");

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
export function parseNFTPayload(payload: Buffer) {
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

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee
export function parseTransferPayload(payload: Buffer) {
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

//This returns a corrected amount, which accounts for the difference between the VAA
//decimals, and the decimals of the asset.
// const normalizeVaaAmount = (
//   amount: bigint,
//   assetDecimals: number
// ): bigint => {
//   const MAX_VAA_DECIMALS = 8;
//   if (assetDecimals <= MAX_VAA_DECIMALS) {
//     return amount;
//   }
//   const decimalStringVaa = formatUnits(amount, MAX_VAA_DECIMALS);
//   const normalizedAmount = parseUnits(decimalStringVaa, assetDecimals);
//   const normalizedBigInt = BigInt(truncate(normalizedAmount.toString(), 0));

//   return normalizedBigInt;
// };

// function truncate(str: string, maxDecimalDigits: number) {
//   if (str.includes(".")) {
//     const parts = str.split(".");
//     return parts[0] + "." + parts[1].slice(0, maxDecimalDigits);
//   }
//   return str;
// }
