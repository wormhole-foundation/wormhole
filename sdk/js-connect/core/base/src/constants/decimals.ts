import { MapLevel, constMap } from "../utils";
import { Platform } from "./platforms";

const nativeDecimalEntries = [
  ["Evm",     18],
  ["Solana",   9],
  ["Sui",      9],
  ["Aptos",    8],
  ["Cosmwasm", 6],
  ["Algorand", 6],
  ["Btc",      8],
  ["Near",    12],
] as const satisfies MapLevel<Platform, number>;

export const nativeDecimals = constMap(nativeDecimalEntries);
