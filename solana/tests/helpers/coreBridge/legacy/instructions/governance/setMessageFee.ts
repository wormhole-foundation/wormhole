import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../../..";
import { BridgeProgramData, Claim, PostedVaaV1 } from "../../state";

export type LegacySetMessageFeeContext = {
  payer: PublicKey;
  bridge?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
};

export function legacySetMessageFeeIx(
  program: CoreBridgeProgram,
  accounts: LegacySetMessageFeeContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsed;

  let { payer, bridge, postedVaa, claim } = accounts;

  if (bridge === undefined) {
    bridge = BridgeProgramData.address(programId);
  }

  if (postedVaa === undefined) {
    postedVaa = PostedVaaV1.address(programId, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      emitterChain,
      Array.from(emitterAddress),
      new BN(sequence.toString())
    );
  }

  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: bridge,
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
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 3);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
