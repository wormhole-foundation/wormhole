import { PublicKey } from "@solana/web3.js";

export { upgradeAuthorityPda, Claim } from "../../../coreBridge/legacy/state";
export * from "./Config";
export * from "./RegisteredEmitter";
export * from "./WrappedAsset";

export const TOKEN_METADATA_PROGRAM_ID = new PublicKey(
  "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
);

export function coreEmitterPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("emitter")], programId)[0];
}

export function custodyAuthorityPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("custody_signer")], programId)[0];
}

export function custodyTokenPda(programId: PublicKey, mint: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([mint.toBuffer()], programId)[0];
}

export function mintAuthorityPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("mint_signer")], programId)[0];
}

export function transferAuthorityPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("authority_signer")], programId)[0];
}

export function wrappedMintPda(
  programId: PublicKey,
  tokenChain: number,
  tokenAddress: number[]
): PublicKey {
  const encodedChain = Buffer.alloc(2);
  encodedChain.writeUInt16BE(tokenChain, 0);
  return PublicKey.findProgramAddressSync(
    [Buffer.from("wrapped"), encodedChain, Buffer.from(tokenAddress)],
    programId
  )[0];
}

export function tokenMetadataPda(mint: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("metadata"), TOKEN_METADATA_PROGRAM_ID.toBuffer(), new PublicKey(mint).toBuffer()],
    TOKEN_METADATA_PROGRAM_ID
  )[0];
}
