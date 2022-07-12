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
  bridgeInfoKey,
  emitterKey,
  emitterSequenceKey,
  feeCollectorKey,
  getEmitterKeys,
  guardianSetKey,
  postedVaaKey,
} from "../accounts";

/** All accounts required to make a cross-program invocation with the Core Bridge program */
export interface PostMessageAccounts {
  bridge: PublicKey;
  message: PublicKey;
  emitter: PublicKey;
  sequence: PublicKey;
  payer: PublicKey;
  feeCollector: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

export function getPostMessageAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  emitterDeriverId: PublicKeyInitData,
  message: PublicKeyInitData
): PostMessageAccounts {
  const { emitter, sequence } = getEmitterKeys(emitterDeriverId, wormholeProgramId);
  console.log("umm", emitter.toString(), sequence.toString());
  console.log("message", new PublicKey(message.toString()).toString());
  return {
    bridge: bridgeInfoKey(wormholeProgramId),
    message: new PublicKey(message),
    emitter,
    sequence,
    payer: new PublicKey(payer),
    feeCollector: feeCollectorKey(wormholeProgramId),
    clock: SYSVAR_CLOCK_PUBKEY,
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
