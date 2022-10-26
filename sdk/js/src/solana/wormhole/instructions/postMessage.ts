import {
  PublicKey,
  PublicKeyInitData,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
} from "@solana/web3.js";
import {
  deriveWormholeBridgeDataKey,
  deriveFeeCollectorKey,
  getEmitterKeys,
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
  emitterProgramId: PublicKeyInitData,
  message: PublicKeyInitData
): PostMessageAccounts {
  const { emitter, sequence } = getEmitterKeys(
    emitterProgramId,
    wormholeProgramId
  );
  return {
    bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    message: new PublicKey(message),
    emitter,
    sequence,
    payer: new PublicKey(payer),
    feeCollector: deriveFeeCollectorKey(wormholeProgramId),
    clock: SYSVAR_CLOCK_PUBKEY,
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
