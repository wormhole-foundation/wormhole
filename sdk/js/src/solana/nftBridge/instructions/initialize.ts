import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { createReadOnlyNftBridgeProgramInterface } from "../program";
import { deriveNftBridgeConfigKey } from "../accounts";

export function createInitializeInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): TransactionInstruction {
  const methods = createReadOnlyNftBridgeProgramInterface(
    nftBridgeProgramId
  ).methods.initialize(wormholeProgramId as any);

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getInitializeAccounts(nftBridgeProgramId, payer) as any,
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
  nftBridgeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData
): InitializeAccounts {
  return {
    payer: new PublicKey(payer),
    config: deriveNftBridgeConfigKey(nftBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
