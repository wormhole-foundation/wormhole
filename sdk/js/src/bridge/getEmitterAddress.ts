import { PublicKeyInitData } from "@solana/web3.js";
import { decodeAddress, getApplicationAddress } from "algosdk";
import { bech32 } from "bech32";
import {
  arrayify,
  BytesLike,
  Hexable,
  sha256,
  zeroPad,
} from "ethers/lib/utils";
import { deriveWormholeEmitterKey } from "../solana/wormhole";
import { uint8ArrayToHex } from "../utils";

export function getEmitterAddressEth(
  contractAddress: number | BytesLike | Hexable
) {
  return Buffer.from(zeroPad(arrayify(contractAddress), 32)).toString("hex");
}

export function getEmitterAddressSolana(programAddress: PublicKeyInitData) {
  return deriveWormholeEmitterKey(programAddress).toBuffer().toString("hex");
}

export async function getEmitterAddressTerra(programAddress: string) {
  return Buffer.from(
    zeroPad(bech32.fromWords(bech32.decode(programAddress).words), 32)
  ).toString("hex");
}

export const getEmitterAddressInjective = getEmitterAddressTerra;

export const getEmitterAddressXpla = getEmitterAddressTerra;

export function getEmitterAddressAlgorand(appId: bigint): string {
  const appAddr: string = getApplicationAddress(appId);
  const decAppAddr: Uint8Array = decodeAddress(appAddr).publicKey;
  const aa: string = uint8ArrayToHex(decAppAddr);
  return aa;
}

export function getEmitterAddressNear(programAddress: string): string {
  return uint8ArrayToHex(arrayify(sha256(Buffer.from(programAddress, "utf8"))));
}
