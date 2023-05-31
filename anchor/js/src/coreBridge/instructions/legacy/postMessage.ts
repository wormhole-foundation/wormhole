import {
  AccountMeta,
  PublicKey,
  PublicKeyInitData,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { ProgramId } from "../../consts";
import { BridgeProgramData, EmitterSequence, FeeCollector } from "../../state";
import { getProgramPubkey } from "../../utils/misc";

/* private */
export type PostMessageConfig = {
  clock: boolean;
  rent: boolean;
  feeCollector: boolean;
};

export class LegacyPostMessageContext {
  bridge: PublicKey;
  message: PublicKey;
  emitter: PublicKey;
  emitterSequence: PublicKey;
  payer: PublicKey;
  feeCollector: PublicKey | null;
  _clock: PublicKey | null;
  _rent: PublicKey | null;
  systemProgram: PublicKey;

  constructor(
    bridge: PublicKeyInitData,
    message: PublicKeyInitData,
    emitter: PublicKeyInitData,
    emitterSequence: PublicKeyInitData,
    payer: PublicKeyInitData,
    feeCollector: PublicKeyInitData | null,
    clock: PublicKeyInitData | null,
    rent: PublicKeyInitData | null,
    systemProgram: PublicKeyInitData
  ) {
    this.bridge = new PublicKey(bridge);
    this.message = new PublicKey(message);
    this.emitter = new PublicKey(emitter);
    this.emitterSequence = new PublicKey(emitterSequence);
    this.payer = new PublicKey(payer);
    this.feeCollector = feeCollector ? new PublicKey(feeCollector) : null;
    this._clock = clock ? new PublicKey(clock) : null;
    this._rent = rent ? new PublicKey(rent) : null;
    this.systemProgram = new PublicKey(systemProgram);
  }

  static new(
    programId: ProgramId,
    message: PublicKeyInitData,
    emitter: PublicKeyInitData,
    payer: PublicKeyInitData,
    config: PostMessageConfig
  ) {
    const bridge = BridgeProgramData.address(programId);
    const emitterSequence = EmitterSequence.address(programId, emitter);
    const feeCollector = config.feeCollector
      ? FeeCollector.address(programId)
      : null;
    const clock = config.clock ? SYSVAR_CLOCK_PUBKEY : null;
    const rent = config.rent ? SYSVAR_RENT_PUBKEY : null;
    const systemProgram = SystemProgram.programId;
    return new LegacyPostMessageContext(
      bridge,
      message,
      emitter,
      emitterSequence,
      payer,
      feeCollector,
      clock,
      rent,
      systemProgram
    );
  }

  static instruction(
    programId: ProgramId,
    message: PublicKeyInitData,
    emitter: PublicKeyInitData,
    payer: PublicKeyInitData,
    args: LegacyPostMessageArgs,
    config?: PostMessageConfig
  ) {
    return legacyPostMessageIx(
      programId,
      LegacyPostMessageContext.new(
        programId,
        message,
        emitter,
        payer,
        config || { clock: false, rent: false, feeCollector: true }
      ),
      args
    );
  }
}

export type LegacyPostMessageArgs = {
  nonce: number;
  payload: Buffer;
  finalityRepr: number;
};

export function legacyPostMessageIx(
  programId: ProgramId,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs
) {
  return handleLegacyPostMessageIx(programId, accounts, args, false);
}

/* private */
export function handleLegacyPostMessageIx(
  programId: ProgramId,
  accounts: LegacyPostMessageContext,
  args: LegacyPostMessageArgs,
  unreliable: boolean
) {
  const thisProgramId = getProgramPubkey(programId);
  const {
    bridge,
    message,
    emitter,
    emitterSequence,
    payer,
    feeCollector,
    _clock,
    _rent,
    systemProgram,
  } = accounts;
  const keys: AccountMeta[] = [
    {
      pubkey: bridge,
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
      pubkey: feeCollector === null ? thisProgramId : feeCollector,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: _clock === null ? thisProgramId : _clock,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: _rent === null ? thisProgramId : _rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: systemProgram,
      isWritable: false,
      isSigner: false,
    },
  ];
  const { nonce, payload, finalityRepr } = args;
  const data = Buffer.alloc(1 + 4 + 4 + payload.length + 1);
  data.writeUInt8(unreliable ? 8 : 1, 0);
  data.writeUInt32LE(nonce, 1);
  data.writeUInt32LE(payload.length, 5);
  data.write(payload.toString("hex"), 9, "hex");
  data.writeUInt8(finalityRepr, 9 + payload.length);

  return new TransactionInstruction({
    keys,
    programId: thisProgramId,
    data,
  });
}
