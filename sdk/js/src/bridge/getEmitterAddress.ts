import { PublicKey } from "@solana/web3.js";
import { decodeAddress, getApplicationAddress } from "algosdk";
import { bech32 } from "bech32";
import { arrayify, BytesLike, Hexable, zeroPad } from "ethers/lib/utils";
import { importTokenWasm } from "../solana/wasm";
import { uint8ArrayToHex } from "../utils";
import { sha256 } from  "js-sha256";

export function getEmitterAddressEth(
  contractAddress: number | BytesLike | Hexable
) {
  return Buffer.from(zeroPad(arrayify(contractAddress), 32)).toString("hex");
}

export async function getEmitterAddressSolana(programAddress: string) {
  const { emitter_address } = await importTokenWasm();
  return Buffer.from(
    zeroPad(new PublicKey(emitter_address(programAddress)).toBytes(), 32)
  ).toString("hex");
}

export async function getEmitterAddressTerra(programAddress: string) {
  return Buffer.from(
    zeroPad(bech32.fromWords(bech32.decode(programAddress).words), 32)
  ).toString("hex");
}

export const getEmitterAddressInjective = getEmitterAddressTerra;

export function getEmitterAddressAlgorand(appId: bigint): string {
  const appAddr: string = getApplicationAddress(appId);
  const decAppAddr: Uint8Array = decodeAddress(appAddr).publicKey;
  const aa: string = uint8ArrayToHex(decAppAddr);
  return aa;
}

export function getEmitterAddressNear(programAddress: string): string {
  return sha256.hex(programAddress);
}
