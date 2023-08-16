import {
  AccountMeta,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { Config, Claim, GuardianSet, PostedVaaV1 } from "../../state";
import { CoreBridgeProgram } from "../../..";
import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";

export type LegacyGuardianSetUpdateContext = {
  payer: PublicKey;
  config?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
  currGuardianSet?: PublicKey;
  newGuardianSet?: PublicKey;
};

export function legacyGuardianSetUpdateIx(
  program: CoreBridgeProgram,
  accounts: LegacyGuardianSetUpdateContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, guardianSetIndex, hash } = parsed;

  let { payer, config, postedVaa, claim, currGuardianSet, newGuardianSet } = accounts;

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

  if (currGuardianSet === undefined) {
    currGuardianSet = GuardianSet.address(programId, guardianSetIndex);
  }

  if (newGuardianSet === undefined) {
    newGuardianSet = GuardianSet.address(programId, guardianSetIndex + 1);
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
      pubkey: currGuardianSet,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: newGuardianSet,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 6);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
