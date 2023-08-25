import { ParsedVaa, parseVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import { PublicKey } from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId, getCoreBridgeProgram } from "../..";
import { Claim, RegisteredEmitter } from "../../legacy/state";
import * as coreBridge from "../../../coreBridge";

export type RegisterChainContext = {
  payer: PublicKey;
  vaa: PublicKey;
  claim?: PublicKey;
  registeredEmitter?: PublicKey;
  legacyRegisteredEmitter?: PublicKey;
  coreBridgeProgram?: PublicKey;
};

export async function registerChainIx(program: TokenBridgeProgram, accounts: RegisterChainContext) {
  const programId = program.programId;

  let { payer, vaa, claim, registeredEmitter, legacyRegisteredEmitter, coreBridgeProgram } =
    accounts;

  const parsed = await coreBridge.EncodedVaa.fetch(getCoreBridgeProgram(program), vaa).then(
    (acct) => parseVaa(acct.buf)
  );
  const { emitterChain, emitterAddress, sequence, payload } = parsed;

  const foreignChain = payload.readUInt16BE(35);
  const foreignEmitter = Array.from(payload.subarray(37));

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (registeredEmitter === undefined) {
    registeredEmitter = RegisteredEmitter.address(programId, foreignChain);
  }

  if (legacyRegisteredEmitter === undefined) {
    legacyRegisteredEmitter = RegisteredEmitter.address(programId, foreignChain, foreignEmitter);
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      Array.from(emitterAddress),
      emitterChain,
      new BN(sequence.toString())
    );
  }

  return program.methods
    .registerChain()
    .accounts({ payer, vaa, claim, registeredEmitter, legacyRegisteredEmitter, coreBridgeProgram })
    .instruction();
}
