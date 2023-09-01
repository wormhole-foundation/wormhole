import {
  CoreBridgeProgram,
  LegacyPostMessageArgs,
  LegacyPostMessageContext,
  handleLegacyPostMessageIx,
} from "../..";

export type LegacyPostMessageUnreliableContext = LegacyPostMessageContext;
export type LegacyPostMessageUnreliableArgs = LegacyPostMessageArgs;

export function legacyPostMessageUnreliableIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostMessageUnreliableContext,
  args: LegacyPostMessageUnreliableArgs
) {
  return handleLegacyPostMessageIx(
    program,
    accounts,
    args,
    true, // unreliable
    {
      emitter: true,
      message: true,
    } // requireOtherSigners
  );
}
