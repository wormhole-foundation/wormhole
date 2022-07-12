import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import {
  isBytes,
  ParsedGovernanceVaa,
  parseGovernanceVaa,
  SignedVaa,
} from "../../../vaa";
import { createReadOnlyWormholeProgramInterface } from "../program";
import {
  deriveWormholeBridgeDataKey,
  deriveClaimKey,
  deriveFeeCollectorKey,
  deriveGuardianSetKey,
  derivePostedVaaKey,
  deriveUpgradeAuthorityKey,
} from "../accounts";
import { BpfLoaderUpgradeable, deriveUpgradeableProgramKey } from "../../utils";

export function createSetFeesInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedGovernanceVaa
): TransactionInstruction {
  const methods =
    createReadOnlyWormholeProgramInterface(wormholeProgramId).methods.setFees();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getSetFeesAccounts(wormholeProgramId, payer, vaa) as any,
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
  vaa: SignedVaa | ParsedGovernanceVaa
): SetFeesAccounts {
  const parsed = isBytes(vaa) ? parseGovernanceVaa(vaa) : vaa;
  return {
    payer: new PublicKey(payer),
    bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      wormholeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    systemProgram: SystemProgram.programId,
  };
}

export function createTransferFeesInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  recipient: PublicKeyInitData,
  vaa: SignedVaa | ParsedGovernanceVaa
): TransactionInstruction {
  const methods =
    createReadOnlyWormholeProgramInterface(
      wormholeProgramId
    ).methods.transferFees();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getTransferFeesAccounts(
      wormholeProgramId,
      payer,
      recipient,
      vaa
    ) as any,
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
  vaa: SignedVaa | ParsedGovernanceVaa
): TransferFeesAccounts {
  const parsed = isBytes(vaa) ? parseGovernanceVaa(vaa) : vaa;
  return {
    payer: new PublicKey(payer),
    bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      wormholeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    feeCollector: deriveFeeCollectorKey(wormholeProgramId),
    recipient: new PublicKey(recipient),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}

export function createUpgradeGuardianSetInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedGovernanceVaa
): TransactionInstruction {
  const methods =
    createReadOnlyWormholeProgramInterface(
      wormholeProgramId
    ).methods.upgradeGuardianSet();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getUpgradeGuardianSetAccounts(
      wormholeProgramId,
      payer,
      vaa
    ) as any,
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
  vaa: SignedVaa | ParsedGovernanceVaa
): UpgradeGuardianSetAccounts {
  const parsed = isBytes(vaa) ? parseGovernanceVaa(vaa) : vaa;
  return {
    payer: new PublicKey(payer),
    bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      wormholeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    guardianSetOld: deriveGuardianSetKey(
      wormholeProgramId,
      parsed.guardianSetIndex
    ),
    guardianSetNew: deriveGuardianSetKey(
      wormholeProgramId,
      parsed.guardianSetIndex + 1
    ),
    systemProgram: SystemProgram.programId,
  };
}

export function createUpgradeContractInstruction(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedGovernanceVaa
): TransactionInstruction {
  const methods =
    createReadOnlyWormholeProgramInterface(
      wormholeProgramId
    ).methods.upgradeContract();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getUpgradeContractAccounts(wormholeProgramId, payer, vaa) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface UpgradeContractAccounts {
  payer: PublicKey;
  bridge: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  upgradeAuthority: PublicKey;
  spill: PublicKey;
  implementation: PublicKey;
  programData: PublicKey;
  wormholeProgram: PublicKey;
  rent: PublicKey;
  clock: PublicKey;
  bpfLoaderUpgradeable: PublicKey;
  systemProgram: PublicKey;
}

export function getUpgradeContractAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedGovernanceVaa,
  spill?: PublicKeyInitData
): UpgradeContractAccounts {
  const parsed = isBytes(vaa) ? parseGovernanceVaa(vaa) : vaa;
  const implementation = parsed.orderPayload;
  if (implementation.length != 32) {
    throw new Error("implementation.length != 32");
  }
  return {
    payer: new PublicKey(payer),
    bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      wormholeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    upgradeAuthority: deriveUpgradeAuthorityKey(wormholeProgramId),
    spill: new PublicKey(spill === undefined ? payer : spill),
    implementation: new PublicKey(implementation),
    programData: deriveUpgradeableProgramKey(wormholeProgramId),
    wormholeProgram: new PublicKey(wormholeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    clock: SYSVAR_CLOCK_PUBKEY,
    bpfLoaderUpgradeable: BpfLoaderUpgradeable.programId,
    systemProgram: SystemProgram.programId,
  };
}
