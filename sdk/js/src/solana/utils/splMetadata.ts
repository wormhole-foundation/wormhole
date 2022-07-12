import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "./account";

export class SplTokenMetadataProgram {
  /**
   * @internal
   */
  constructor() {}

  /**
   * Public key that identifies the SPL Token Metadata program
   */
  static programId: PublicKey = new PublicKey(
    "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
  );
}

export function deriveSplTokenMetaKey(mint: PublicKeyInitData): PublicKey {
  return deriveAddress(
    [
      Buffer.from("metadata"),
      SplTokenMetadataProgram.programId.toBuffer(),
      new PublicKey(mint).toBuffer(),
    ],
    SplTokenMetadataProgram.programId
  );
}
