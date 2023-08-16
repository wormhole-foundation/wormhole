import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";
import { Config, FeeCollector, EmitterSequence } from "../state";

export enum Finality {
  Confirmed,
  Finalized,
}

export type LegacyPostMessageContext = {
  config?: PublicKey;
  message: PublicKey;
  emitter: PublicKey;
  emitterSequence?: PublicKey;
  payer: PublicKey;
  feeCollector?: PublicKey;
  clock?: PublicKey;
  rent?: PublicKey;
};

export type LegacyPostMessageArgs = {
  nonce: number;
  payload: Buffer;
  finality: Finality;
};

export function legacyPostMessageIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs
) {
  return handleLegacyPostMessageIx(program, accounts, args, false);
}

/* private */
export function handleLegacyPostMessageIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs,
  unreliable: boolean
) {
  const programId = program.programId;

  let { config, message, emitter, emitterSequence, payer, feeCollector, clock, rent } = accounts;

  if (config === undefined) {
    config = Config.address(program.programId);
  }

  if (emitterSequence === undefined) {
    emitterSequence = EmitterSequence.address(program.programId, emitter);
  }

  if (feeCollector === undefined) {
    feeCollector = FeeCollector.address(program.programId);
  } else if (feeCollector === null) {
    feeCollector = programId;
  }

  if (clock === undefined) {
    clock = SYSVAR_CLOCK_PUBKEY;
  } else if (clock === null) {
    clock = programId;
  }

  if (rent === undefined) {
    rent = SYSVAR_CLOCK_PUBKEY;
  } else if (rent === null) {
    rent = programId;
  }

  const keys: AccountMeta[] = [
    {
      pubkey: config,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: message,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: emitter,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: emitterSequence,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: feeCollector,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: clock,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: rent,
      isWritable: false,
      isSigner: false,
    },
  ];
  const { nonce, payload, finality } = args;
  const data = Buffer.alloc(1 + 4 + 4 + payload.length + 1);
  data.writeUInt8(unreliable ? 8 : 1, 0);
  data.writeUInt32LE(nonce, 1);
  data.writeUInt32LE(payload.length, 5);
  data.set(payload, 9);
  data.writeUInt8(finality, 9 + payload.length);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
