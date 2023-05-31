import {
  PublicKeyInitData,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
} from "@solana/web3.js";
import { ProgramId } from "../../consts";
import {
  LegacyPostMessageArgs,
  LegacyPostMessageContext,
  PostMessageConfig,
  handleLegacyPostMessageIx,
} from "./postMessage";
import { BridgeProgramData, EmitterSequence, FeeCollector } from "../../state";

export class LegacyPostMessageUnreliableContext extends LegacyPostMessageContext {
  private constructor(
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
    super(
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

  static new(
    programId: ProgramId,
    message: PublicKeyInitData,
    emitter: PublicKeyInitData,
    payer: PublicKeyInitData,
    config: PostMessageConfig
  ) {
    return super.new(
      programId,
      message,
      emitter,
      payer,
      config
    ) as LegacyPostMessageUnreliableContext;
  }

  static instruction(
    programId: ProgramId,
    message: PublicKeyInitData,
    emitter: PublicKeyInitData,
    payer: PublicKeyInitData,
    args: LegacyPostMessageUnreliableArgs,
    config?: PostMessageConfig
  ) {
    return legacyPostMessageUnreliableIx(
      programId,
      LegacyPostMessageUnreliableContext.new(
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

export type LegacyPostMessageUnreliableArgs = LegacyPostMessageArgs;

export function legacyPostMessageUnreliableIx(
  programId: ProgramId,
  accounts: LegacyPostMessageUnreliableContext,
  args: LegacyPostMessageUnreliableArgs
) {
  return handleLegacyPostMessageIx(programId, accounts, args, true);
}
