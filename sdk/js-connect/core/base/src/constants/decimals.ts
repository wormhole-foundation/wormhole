import { constMap } from "../utils";
import { PlatformName } from "./platforms";

const nativeDecimalEntries = [
  ["Evm", 18n],
  ["Solana", 9n],
  ["Sui", 9n],
  ["Aptos", 8n],
  ["Cosmwasm", 6n],
  ["Algorand", 6n],
  ["Btc", 8n],
  ["Near", 12n],
] as const satisfies readonly (readonly [PlatformName, bigint])[];

export const nativeDecimals = constMap(nativeDecimalEntries);
