import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export function deriveFeeCollectorKey(
  wormholeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("fee_collector")], wormholeProgramId);
}
