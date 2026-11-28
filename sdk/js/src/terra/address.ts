import { zeroPad } from "@ethersproject/bytes";

export const isNativeTerra = (string = "") =>
  string.startsWith("u") && string.length === 4;

export const isNativeDenom = (string = "") =>
  isNativeTerra(string) || string === "uluna";

export function buildNativeId(denom: string): Uint8Array {
  const bytes = [];
  for (let i = 0; i < denom.length; i++) {
    bytes.push(denom.charCodeAt(i));
  }
  const padded = zeroPad(new Uint8Array(bytes), 32);
  padded[0] = 1;
  return padded;
}
