import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../../..";
import { Config, Claim, PostedVaaV1, feeCollectorPda } from "../../state";

export type LegacyTransferFeesContext = {
  payer: PublicKey;
  config?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
  feeCollector?: PublicKey;
  recipient: PublicKey;
  rent?: PublicKey;
};

export function legacyTransferFeesIx(
  program: CoreBridgeProgram,
  accounts: LegacyTransferFeesContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash } = parsed;

  let { payer, config, postedVaa, claim, feeCollector, recipient, rent } = accounts;

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

  if (feeCollector === undefined) {
    feeCollector = feeCollectorPda(programId);
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
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
      pubkey: feeCollector,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: recipient,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 4);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
