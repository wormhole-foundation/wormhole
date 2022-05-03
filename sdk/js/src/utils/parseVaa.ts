import { BigNumber } from "@ethersproject/bignumber";
import { ChainId } from "./consts";

export const METADATA_REPLACE = new RegExp("\u0000", "g");

// TODO: remove `as ChainId` in next minor version as we can't ensure it will match our type definition

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
export const parseNFTPayload = (arr: Buffer) => {
  const originAddress = arr.slice(1, 1 + 32).toString("hex");
  const originChain = arr.readUInt16BE(33) as ChainId;
  const symbol = Buffer.from(arr.slice(35, 35 + 32))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const name = Buffer.from(arr.slice(67, 67 + 32))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const tokenId = BigNumber.from(arr.slice(99, 99 + 32));
  const uri_len = arr.readUInt8(131);
  const uri = Buffer.from(arr.slice(132, 132 + uri_len))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const target_offset = 132 + uri_len;
  const targetAddress = arr
    .slice(target_offset, target_offset + 32)
    .toString("hex");
  const targetChain = arr.readUInt16BE(target_offset + 32) as ChainId;
  return {
    originAddress,
    originChain,
    symbol,
    name,
    tokenId,
    uri,
    targetAddress,
    targetChain,
  };
};

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee
export const parseTransferPayload = (arr: Buffer) => ({
  amount: BigNumber.from(arr.slice(1, 1 + 32)).toBigInt(),
  originAddress: arr.slice(33, 33 + 32).toString("hex"),
  originChain: arr.readUInt16BE(65) as ChainId,
  targetAddress: arr.slice(67, 67 + 32).toString("hex"),
  targetChain: arr.readUInt16BE(99) as ChainId,
  fee: BigNumber.from(arr.slice(101, 101 + 32)).toBigInt(),
});

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
