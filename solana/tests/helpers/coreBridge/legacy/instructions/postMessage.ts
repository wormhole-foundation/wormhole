import {
  AccountMeta,
  Commitment,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram, toLegacyCommitment } from "../..";
import { Config, EmitterSequence, feeCollectorPda } from "../state";

export type LegacyPostMessageContext = {
  config?: PublicKey;
  message: PublicKey;
  emitter: PublicKey | null;
  emitterSequence?: PublicKey;
  payer: PublicKey;
  feeCollector?: PublicKey | null;
  clock?: PublicKey | null;
  rent?: PublicKey | null;
};

export function legacyPostMessageAccounts(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageContext
): LegacyPostMessageContext {
  const programId = program.programId;

  let { config, message, emitter, emitterSequence, payer, feeCollector, clock, rent } = accounts;
  if (config === undefined) {
    config = Config.address(program.programId);
  }

  if (emitter === null) {
    emitter = programId;
    if (emitterSequence === undefined) {
      throw new Error("emitterSequence must be defined if emitter is null");
    }
  }

  if (emitterSequence === undefined) {
    emitterSequence = EmitterSequence.address(program.programId, emitter);
  }

  if (feeCollector === undefined) {
    feeCollector = feeCollectorPda(program.programId);
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

  return {
    config,
    message,
    emitter,
    emitterSequence,
    payer,
    feeCollector,
    clock,
    rent,
  };
}

export type LegacyPostMessageArgs = {
  nonce: number;
  payload: Buffer;
  commitment: Commitment;
};

export function legacyPostMessageIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs,
  requireOtherSigners: {
    message?: boolean;
  } = {}
) {
  return handleLegacyPostMessageIx(program, accounts, args, false, requireOtherSigners);
}

/* private */
export function handleLegacyPostMessageIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs,
  unreliable: boolean,
  requireOtherSigners: {
    message?: boolean;
  }
) {
  const { config, message, emitter, emitterSequence, payer, feeCollector, clock, rent } =
    legacyPostMessageAccounts(program, accounts);

  let { message: messageIsSigner } = requireOtherSigners;

  const emitterIsSigner = emitter!.equals(program.programId) ? false : true;

  if (messageIsSigner === undefined) {
    messageIsSigner = true;
  }

  const keys: AccountMeta[] = [
    {
      pubkey: config!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: message!,
      isWritable: true,
      isSigner: messageIsSigner,
    },
    {
      pubkey: emitter!,
      isWritable: false,
      isSigner: emitterIsSigner,
    },
    {
      pubkey: emitterSequence!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: feeCollector!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: clock!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: rent!,
      isWritable: false,
      isSigner: false,
    },
  ];
  const { nonce, payload, commitment } = args;
  const data = Buffer.alloc(1 + 4 + 4 + payload.length + 1);
  data.writeUInt8(unreliable ? 8 : 1, 0);
  data.writeUInt32LE(nonce, 1);
  data.writeUInt32LE(payload.length, 5);
  data.set(payload, 9);
  data.writeUInt8(toLegacyCommitment(commitment), 9 + payload.length);

  return new TransactionInstruction({
    keys,
    programId: program.programId,
    data,
  });
}
