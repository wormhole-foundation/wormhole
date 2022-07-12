import {
  PublicKey,
  AccountMeta,
  AccountInfo,
  PublicKeyInitData,
} from "@solana/web3.js";

/**
 * Find valid program address. See {@link PublicKey.findProgramAddressSync} for details.
 *
 * @param {(Buffer | Uint8Array)[]} seeds - seeds for PDA
 * @param {PublicKeyInitData} programId - program address
 * @returns PDA
 */
export function deriveAddress(
  seeds: (Buffer | Uint8Array)[],
  programId: PublicKeyInitData
): PublicKey {
  return PublicKey.findProgramAddressSync(seeds, new PublicKey(programId))[0];
}

/**
 * Factory to create AccountMeta with `isWritable` set to `true`
 *
 * @param {PublicKEyInitData} pubkey - account address
 * @param {boolean} isSigner - whether account authorized transaction
 * @returns metadata for writable account
 */
export function newAccountMeta(
  pubkey: PublicKeyInitData,
  isSigner: boolean
): AccountMeta {
  return {
    pubkey: new PublicKey(pubkey),
    isWritable: true,
    isSigner,
  };
}

/**
 * Factory to create AccountMeta with `isWritable` set to `false`
 *
 * @param {PublicKEyInitData} pubkey - account address
 * @param {boolean} isSigner - whether account authorized transaction
 * @returns metadata for read-only account
 */
export function newReadOnlyAccountMeta(
  pubkey: PublicKeyInitData,
  isSigner: boolean
): AccountMeta {
  return {
    pubkey: new PublicKey(pubkey),
    isWritable: false,
    isSigner,
  };
}

/**
 * Get serialized data from account
 *
 * @param {AccountInfo<Buffer>} info - Solana AccountInfo
 * @returns serialized data as Buffer
 */
export function getAccountData(info: AccountInfo<Buffer> | null): Buffer {
  if (info === null) {
    throw Error("account info is null");
  }
  return info.data;
}
