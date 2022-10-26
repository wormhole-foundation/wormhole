import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export function deriveSenderAccountKey(
  cpiProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("sender")], cpiProgramId);
}

export function deriveRedeemerAccountKey(
  cpiProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("redeemer")], cpiProgramId);
}
