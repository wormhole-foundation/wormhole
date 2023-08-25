import { PublicKey } from "@solana/web3.js";
import { TokenBridgeProgram } from "../..";
import { RegisteredEmitter } from "../../legacy/state";

export type SecureRegisteredEmitterContext = {
  payer: PublicKey;
  registeredEmitter?: PublicKey;
  legacyRegisteredEmitter: PublicKey;
};

export type SecureRegisteredEmitterDirective =
  | {
      init: {};
    }
  | {
      closeLegacy: {};
    };

export async function secureRegisteredEmitterIx(
  program: TokenBridgeProgram,
  accounts: SecureRegisteredEmitterContext,
  directive: SecureRegisteredEmitterDirective
) {
  const programId = program.programId;

  let { payer, registeredEmitter, legacyRegisteredEmitter } = accounts;

  if (registeredEmitter === undefined) {
    const foreignChain = await RegisteredEmitter.fromAccountAddress(
      program.provider.connection,
      legacyRegisteredEmitter
    ).then((emitter) => emitter.chain);
    registeredEmitter = RegisteredEmitter.address(programId, foreignChain);
  }

  return program.methods
    .secureRegisteredEmitter(directive)
    .accounts({ payer, registeredEmitter, legacyRegisteredEmitter })
    .instruction();
}
