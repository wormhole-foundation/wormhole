import { base16, base64, base58 } from '@scure/base';

export const stripPrefix = (prefix: string, str: string): string =>
  str.startsWith(prefix) ? str.slice(prefix.length) : str;

const isHexRegex = /^(?:0x)?[0-9a-fA-F]+$/;
export const hex = {
  valid: (input: string) => isHexRegex.test(input),
  decode: (input: string) => base16.decode(stripPrefix("0x", input).toUpperCase()),
  encode: (input: string | Uint8Array, prefix: boolean = false) => {
    input = typeof input === "string" ? toUint8Array(input) : input;
    return (prefix ? "0x" : "") + base16.encode(input).toLowerCase()
  }
}

// regex string to check if the input could possibly be base64 encoded.
// WARNING: There are clear text strings that are NOT base64 encoded that will pass this check.
const isB64Regex = /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;
export const b64 = {
  valid: (input: string) => isB64Regex.test(input),
  decode: base64.decode,
  encode: (input: string | Uint8Array) =>
    base64.encode(typeof input === "string" ? toUint8Array(input) : input)
}

export const b58 = {
  decode: base58.decode,
  encode: (input: string | Uint8Array) =>
    base58.encode(typeof input === "string" ? toUint8Array(input) : input),
}

export const bignum = {
  decode: (input: string) => BigInt(input),
  encode: (input: bigint, prefix: boolean = false) => (prefix ? "0x" : "") + input.toString(16)
}

export const toUint8Array = (value: string | bigint): Uint8Array =>
  typeof value === "bigint"
  ? toUint8Array(bignum.encode(value))
  : (new TextEncoder()).encode(value);

export const fromUint8Array = (value: Uint8Array): string =>
  (new TextDecoder()).decode(value);
