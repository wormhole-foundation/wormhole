import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "./account";

export class BpfLoaderUpgradeable {
  /**
   * @internal
   */
  constructor() {}

  /**
   * Public key that identifies the SPL Token Metadata program
   */
  static programId: PublicKey = new PublicKey(
    "BPFLoaderUpgradeab1e11111111111111111111111"
  );
}

export function deriveUpgradeableProgramKey(programId: PublicKeyInitData) {
  return deriveAddress(
    [new PublicKey(programId).toBuffer()],
    BpfLoaderUpgradeable.programId
  );
}
