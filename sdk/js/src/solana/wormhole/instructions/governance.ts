import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { isBytes, ParsedVaa, parseVaa, SignedVaa } from "../../../vaa/wormhole";
import { createReadOnlyWormholeProgramInterface } from "../program";
import { bridgeInfoKey, claimKey, feeCollectorKey, guardianSetKey, postedVaaKey } from "../accounts";

export function createSetFeesInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedVaa
): TransactionInstruction {
  const parsed = isBytes(vaa) ? parseVaa(vaa) : vaa;
  const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId).methods.setFees();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getSetFeesAccounts(wormholeProgramId, payer, parsed) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface SetFeesAccounts {
  payer: PublicKey;
  bridge: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  systemProgram: PublicKey;
}

export function getSetFeesAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  parsed: ParsedVaa
): SetFeesAccounts {
  return {
    payer: new PublicKey(payer),
    bridge: bridgeInfoKey(wormholeProgramId),
    vaa: postedVaaKey(wormholeProgramId, parsed.hash),
    claim: claimKey(wormholeProgramId, parsed.emitterAddress, parsed.emitterChain, parsed.sequence),
    systemProgram: SystemProgram.programId,
  };
}

export function createTransferFeesInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  recipient: PublicKeyInitData,
  vaa: SignedVaa | ParsedVaa
): TransactionInstruction {
  const parsed = isBytes(vaa) ? parseVaa(vaa) : vaa;
  const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId).methods.transferFees();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getTransferFeesAccounts(wormholeProgramId, payer, recipient, parsed) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface TransferFeesAccounts {
  payer: PublicKey;
  bridge: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  feeCollector: PublicKey;
  recipient: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

export function getTransferFeesAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  recipient: PublicKeyInitData,
  parsed: ParsedVaa
): TransferFeesAccounts {
  return {
    payer: new PublicKey(payer),
    bridge: bridgeInfoKey(wormholeProgramId),
    vaa: postedVaaKey(wormholeProgramId, parsed.hash),
    claim: claimKey(wormholeProgramId, parsed.emitterAddress, parsed.emitterChain, parsed.sequence),
    feeCollector: feeCollectorKey(wormholeProgramId),
    recipient: new PublicKey(recipient),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}

export function createUpgradeGuardianSetInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedVaa
): TransactionInstruction {
  const parsed = isBytes(vaa) ? parseVaa(vaa) : vaa;
  const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId).methods.upgradeGuardianSet();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getUpgradeGuardianSetAccounts(wormholeProgramId, payer, parsed) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface UpgradeGuardianSetAccounts {
  payer: PublicKey;
  bridge: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  guardianSetOld: PublicKey;
  guardianSetNew: PublicKey;
  systemProgram: PublicKey;
}

export function getUpgradeGuardianSetAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  parsed: ParsedVaa
): UpgradeGuardianSetAccounts {
  return {
    payer: new PublicKey(payer),
    bridge: bridgeInfoKey(wormholeProgramId),
    vaa: postedVaaKey(wormholeProgramId, parsed.hash),
    claim: claimKey(wormholeProgramId, parsed.emitterAddress, parsed.emitterChain, parsed.sequence),
    guardianSetOld: guardianSetKey(wormholeProgramId, parsed.guardianSetIndex),
    guardianSetNew: guardianSetKey(wormholeProgramId, parsed.guardianSetIndex + 1),
    systemProgram: SystemProgram.programId,
  };
}
