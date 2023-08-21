import {
  AccountMeta,
  PublicKey,
  SYSVAR_INSTRUCTIONS_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type LegacyVerifySignaturesContext = {
  payer: PublicKey;
  guardianSet: PublicKey;
  signatureSet: PublicKey;
  instructions?: PublicKey;
  rent?: PublicKey;
};

export type LegacyVerifySignaturesArgs = {
  signerIndices: number[];
};

export function legacyVerifySignaturesIx(
  program: CoreBridgeProgram,
  accounts: LegacyVerifySignaturesContext,
  args: LegacyVerifySignaturesArgs
) {
  const programId = program.programId;
  let { payer, guardianSet, signatureSet, instructions, rent } = accounts;

  if (instructions === undefined) {
    instructions = SYSVAR_INSTRUCTIONS_PUBKEY;
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
      pubkey: guardianSet,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: signatureSet,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: instructions,
      isWritable: false,
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

  const { signerIndices } = args;
  const numSigners = signerIndices.length;
  const data = Buffer.alloc(1 + numSigners);
  data.writeUInt8(7, 0);
  for (let i = 0; i < numSigners; ++i) {
    data.writeInt8(signerIndices[i], i + 1);
  }

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
