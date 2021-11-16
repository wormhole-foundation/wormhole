import { bech32 } from "bech32";
import { ADDRESS_PREFIX, OPERATOR_PREFIX } from "./consts";

export function fromAccAddress(address: string): BinaryAddress {
  return { bytes: Buffer.from(bech32.fromWords(bech32.decode(address).words)) };
}

export function fromValAddress(valAddress: string): BinaryAddress {
  return {
    bytes: Buffer.from(bech32.fromWords(bech32.decode(valAddress).words)),
  };
}

export function fromBase64(address: string): BinaryAddress {
  return { bytes: Buffer.from(address, "base64") };
}

export function toAccAddress(address: BinaryAddress): string {
  return bech32.encode(ADDRESS_PREFIX, bech32.toWords(address.bytes));
}

export function toValAddress(address: BinaryAddress): string {
  return bech32.encode(OPERATOR_PREFIX, bech32.toWords(address.bytes));
}

export function toBase64(address: BinaryAddress): string {
  return Buffer.from(address.bytes).toString("base64");
}

type BinaryAddress = {
  bytes: Uint8Array;
};
