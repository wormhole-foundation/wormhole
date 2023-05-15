import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "./account";

export const TOKEN_METADATA_PROGRAM_ID = new PublicKey(
  "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
);

export function deriveTokenMetadataKey(mint: PublicKeyInitData): PublicKey {
  return deriveAddress(
    [
      Buffer.from("metadata"),
      TOKEN_METADATA_PROGRAM_ID.toBuffer(),
      new PublicKey(mint).toBuffer(),
    ],
    TOKEN_METADATA_PROGRAM_ID
  );
}
