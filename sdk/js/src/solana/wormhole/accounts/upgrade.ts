import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils/account";

export function upgradeKey(wormholeProgramId: PublicKeyInitData): PublicKey {
  return deriveAddress([Buffer.from("upgrade")], wormholeProgramId);
}
