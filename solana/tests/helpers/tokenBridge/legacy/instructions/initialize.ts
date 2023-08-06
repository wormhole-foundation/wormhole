import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram } from "../..";
import { Config } from "../state";

export type LegacyInitializeContext = {
  payer: PublicKey;
  config?: PublicKey;
};

export type LegacyInitializeArgs = {
  coreBridgeProgram: PublicKey;
};

export function legacyInitializeIx(
  program: TokenBridgeProgram,
  accounts: LegacyInitializeContext,
  args: LegacyInitializeArgs
) {
  const programId = program.programId;

  let { payer, config } = accounts;

  if (config === undefined) {
    config = Config.address(programId);
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
      pubkey: SYSVAR_RENT_PUBKEY,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];

  const { coreBridgeProgram } = args;
  const data = Buffer.alloc(1 + 32);
  data.writeUInt8(0, 0);
  data.set(coreBridgeProgram.toBuffer(), 1);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
