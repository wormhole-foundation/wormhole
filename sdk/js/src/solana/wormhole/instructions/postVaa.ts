import {
  PublicKey,
  PublicKeyInitData,
  TransactionInstruction,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
} from "@solana/web3.js";
import { createReadOnlyWormholeProgramInterface } from "../program";
import {
  deriveWormholeInfoKey,
  deriveGuardianSetKey,
  derivePostedVaaKey,
} from "../accounts";
import { isBytes, ParsedVaa, parseVaa, SignedVaa } from "../../../vaa";

/**
 * Make {@link TransactionInstruction} for `post_vaa` instruction.
 *
 * This is used in {@link createPostSignedVaaTransactions}'s last transaction.
 * `signatureSet` is a {@link web3.Keypair} generated outside of this method, which was used
 * to write signatures and the message hash to.
 *
 * https://github.com/certusone/wormhole/blob/dev.v2/solana/bridge/program/src/api/post_vaa.rs
 *
 * @param {PublicKeyInitData} wormholeProgramId - wormhole program address
 * @param {PublicKeyInitData} payer - transaction signer address
 * @param {SignedVaa | ParsedVaa} vaa - either signed VAA bytes or parsed VAA (use {@link parseVaa} on signed VAA)
 * @param {PublicKeyInitData} signatureSet - key for signature set account
 */
export function createPostVaaInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedVaa,
  signatureSet: PublicKeyInitData
): TransactionInstruction {
  const parsed = isBytes(vaa) ? parseVaa(vaa) : vaa;
  const methods = createReadOnlyWormholeProgramInterface(
    wormholeProgramId
  ).methods.postVaa(
    parsed.version,
    parsed.guardianSetIndex,
    parsed.timestamp,
    parsed.nonce,
    parsed.emitterChain,
    parsed.emitterAddress as any,
    parsed.sequence as any,
    parsed.consistencyLevel,
    parsed.payload
  );

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getPostVaaAccounts(
      wormholeProgramId,
      payer,
      signatureSet,
      parsed
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface PostVaaAccounts {
  guardianSet: PublicKey;
  bridge: PublicKey;
  signatureSet: PublicKey;
  vaa: PublicKey;
  payer: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

export function getPostVaaAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  signatureSet: PublicKeyInitData,
  vaa: SignedVaa | ParsedVaa
): PostVaaAccounts {
  const parsed = isBytes(vaa) ? parseVaa(vaa) : vaa;
  return {
    guardianSet: deriveGuardianSetKey(
      wormholeProgramId,
      parsed.guardianSetIndex
    ),
    bridge: deriveWormholeInfoKey(wormholeProgramId),
    signatureSet: new PublicKey(signatureSet),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    payer: new PublicKey(payer),
    clock: SYSVAR_CLOCK_PUBKEY,
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
