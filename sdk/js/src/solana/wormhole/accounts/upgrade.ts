import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export function deriveUpgradeAuthorityKey(
  wormholeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("upgrade")], wormholeProgramId);
}
