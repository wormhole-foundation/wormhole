import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { createReadOnlyTokenBridgeProgramInterface } from "../program";
import { deriveTokenBridgeConfigKey } from "../accounts";

export function createInitializeInstruction(
  tokenBridgeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): TransactionInstruction {
  const methods = createReadOnlyTokenBridgeProgramInterface(
    tokenBridgeProgramId
  ).methods.initialize(wormholeProgramId as any);

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getInitializeAccounts(tokenBridgeProgramId, payer) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface InitializeAccounts {
  payer: PublicKey;
  config: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

export function getInitializeAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData
): InitializeAccounts {
  return {
    payer: new PublicKey(payer),
    config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
