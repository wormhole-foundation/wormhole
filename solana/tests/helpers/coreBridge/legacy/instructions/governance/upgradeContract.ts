import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../../..";
import { BPF_LOADER_UPGRADEABLE_PROGRAM_ID, ProgramData } from "../../../../native";
import { Config, Claim, PostedVaaV1, upgradeAuthorityPda } from "../../state";

export type LegacyUpgradeContractContext = {
  payer: PublicKey;
  config?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
  upgradeAuthority?: PublicKey;
  spill?: PublicKey;
  buffer?: PublicKey;
  programData?: PublicKey;
  thisProgram?: PublicKey;
  rent?: PublicKey;
  clock?: PublicKey;
  bpfLoaderUpgradeableProgram?: PublicKey;
};

export function legacyUpgradeContractIx(
  program: CoreBridgeProgram,
  accounts: LegacyUpgradeContractContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsed;

  let {
    payer,
    config,
    postedVaa,
    claim,
    upgradeAuthority,
    spill,
    buffer,
    programData,
    thisProgram,
    rent,
    clock,
    bpfLoaderUpgradeableProgram,
  } = accounts;

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (postedVaa === undefined) {
    postedVaa = PostedVaaV1.address(programId, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      Array.from(emitterAddress),
      emitterChain,
      new BN(sequence.toString())
    );
  }

  if (upgradeAuthority === undefined) {
    upgradeAuthority = upgradeAuthorityPda(programId);
  }

  if (spill === undefined) {
    spill = payer;
  }

  if (buffer === undefined) {
    buffer = new PublicKey(parsed.payload.subarray(-32));
  }

  if (programData === undefined) {
    programData = ProgramData.address(programId);
  }

  if (thisProgram === undefined) {
    thisProgram = programId;
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
  }

  if (clock === undefined) {
    clock = SYSVAR_CLOCK_PUBKEY;
  }

  if (bpfLoaderUpgradeableProgram === undefined) {
    bpfLoaderUpgradeableProgram = BPF_LOADER_UPGRADEABLE_PROGRAM_ID;
  }

  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: config,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: postedVaa,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: claim,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: upgradeAuthority,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: spill,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: buffer,
      isWritable: true, // legacy requires this to be writable, but the rewrite does not
      isSigner: false,
    },
    {
      pubkey: programData,
      isWritable: true, // legacy requires this to be writable, but the rewrite does not
      isSigner: false,
    },
    {
      pubkey: thisProgram,
      isWritable: true, // legacy requires this to be writable, but the rewrite does not
      isSigner: false,
    },
    {
      pubkey: rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: clock,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: bpfLoaderUpgradeableProgram,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 5);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
