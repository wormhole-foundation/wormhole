import { PublicKey } from "@solana/web3.js";

export * from "./Config";
export const TOKEN_METADATA_PROGRAM_ID = new PublicKey(
  "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
);

export function coreEmitterPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("emitter")],
    programId
  )[0];
}

export function wrappedAssetPda(
  programId: PublicKey,
  mint: PublicKey
): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("meta"), mint.toBuffer()],
    programId
  )[0];
}

export function wrappedMintPda(
  programId: PublicKey,
  tokenChain: number,
  tokenAddress: number[]
): PublicKey {
  const encodedChain = Buffer.alloc(2);
  encodedChain.writeUInt16BE(tokenChain, 0);
  return PublicKey.findProgramAddressSync(
    [Buffer.from("meta"), encodedChain, Buffer.from(tokenAddress)],
    programId
  )[0];
}

export function tokenMetadataPda(mint: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [
      Buffer.from("metadata"),
      TOKEN_METADATA_PROGRAM_ID.toBuffer(),
      new PublicKey(mint).toBuffer(),
    ],
    TOKEN_METADATA_PROGRAM_ID
  )[0];
}
