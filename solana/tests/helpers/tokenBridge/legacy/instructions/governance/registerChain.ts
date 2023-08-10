import { ParsedVaa } from "@certusone/wormhole-sdk";
import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import * as coreBridge from "../../../../coreBridge";
import { Claim, Config, RegisteredEmitter } from "../../state";

export type LegacyRegisterChainContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  registeredEmitter?: PublicKey;
  postedVaa?: PublicKey;
  claim?: PublicKey;
  rent?: PublicKey; // TODO: demonstrate this isn't needed in tests
  coreBridgeProgram?: PublicKey;
};

export function legacyRegisterChainIx(
  program: TokenBridgeProgram,
  accounts: LegacyRegisterChainContext,
  parsed: ParsedVaa
) {
  const programId = program.programId;
  const { emitterChain, emitterAddress, sequence, hash, payload } = parsed;

  const foreignChain = payload.readUInt16BE(35);
  const foreignEmitter = Array.from(payload.subarray(37));

  let { payer, config, registeredEmitter, postedVaa, claim, rent, coreBridgeProgram } = accounts;

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (registeredEmitter === undefined) {
    registeredEmitter = RegisteredEmitter.address(programId, foreignChain, foreignEmitter);
  }

  if (postedVaa === undefined) {
    postedVaa = coreBridge.PostedVaaV1.address(coreBridgeProgram, Array.from(hash));
  }

  if (claim === undefined) {
    claim = Claim.address(
      programId,
      Array.from(emitterAddress),
      emitterChain,
      new BN(sequence.toString())
    );
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
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: registeredEmitter,
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
      pubkey: rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreBridgeProgram,
      isWritable: false,
      isSigner: false,
    },
  ];

  const data = Buffer.alloc(1, 6);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
