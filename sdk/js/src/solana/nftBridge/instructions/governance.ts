import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { createReadOnlyNftBridgeProgramInterface } from "../program";
import { deriveClaimKey, derivePostedVaaKey } from "../../wormhole";
import {
  deriveEndpointKey,
  deriveNftBridgeConfigKey,
  deriveUpgradeAuthorityKey,
} from "../accounts";
import {
  isBytes,
  ParsedNftBridgeRegisterChainVaa,
  ParsedNftBridgeUpgradeContractVaa,
  parseNftBridgeRegisterChainVaa,
  parseNftBridgeUpgradeContractVaa,
  SignedVaa,
} from "../../../vaa";
import { BpfLoaderUpgradeable, deriveUpgradeableProgramKey } from "../../utils";

export function createRegisterChainInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftBridgeRegisterChainVaa
): TransactionInstruction {
  const methods =
    createReadOnlyNftBridgeProgramInterface(
      nftBridgeProgramId
    ).methods.registerChain();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getRegisterChainAccounts(
      nftBridgeProgramId,
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

export interface RegisterChainAccounts {
  payer: PublicKey;
  config: PublicKey;
  endpoint: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getRegisterChainAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftBridgeRegisterChainVaa
): RegisterChainAccounts {
  const parsed = isBytes(vaa) ? parseNftBridgeRegisterChainVaa(vaa) : vaa;
  return {
    payer: new PublicKey(payer),
    config: deriveNftBridgeConfigKey(nftBridgeProgramId),
    endpoint: deriveEndpointKey(
      nftBridgeProgramId,
      parsed.foreignChain,
      parsed.foreignAddress
    ),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      nftBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}

export function createUpgradeContractInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftBridgeUpgradeContractVaa,
  spill?: PublicKeyInitData
): TransactionInstruction {
  const methods =
    createReadOnlyNftBridgeProgramInterface(
      nftBridgeProgramId
    ).methods.upgradeContract();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getUpgradeContractAccounts(
      nftBridgeProgramId,
      wormholeProgramId,
      payer,
      vaa,
      spill
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface UpgradeContractAccounts {
  payer: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  upgradeAuthority: PublicKey;
  spill: PublicKey;
  implementation: PublicKey;
  programData: PublicKey;
  nftBridgeProgram: PublicKey;
  rent: PublicKey;
  clock: PublicKey;
  bpfLoaderUpgradeable: PublicKey;
  systemProgram: PublicKey;
}

export function getUpgradeContractAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftBridgeUpgradeContractVaa,
  spill?: PublicKeyInitData
): UpgradeContractAccounts {
  const parsed = isBytes(vaa) ? parseNftBridgeUpgradeContractVaa(vaa) : vaa;
  return {
    payer: new PublicKey(payer),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      nftBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    upgradeAuthority: deriveUpgradeAuthorityKey(nftBridgeProgramId),
    spill: new PublicKey(spill === undefined ? payer : spill),
    implementation: new PublicKey(parsed.newContract),
    programData: deriveUpgradeableProgramKey(nftBridgeProgramId),
    nftBridgeProgram: new PublicKey(nftBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    clock: SYSVAR_CLOCK_PUBKEY,
    bpfLoaderUpgradeable: BpfLoaderUpgradeable.programId,
    systemProgram: SystemProgram.programId,
  };
}
