import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";
import { Config, GuardianSet, PostedVaaV1 } from "../state";
import { ethers } from "ethers";

export type LegacyPostVaaContext = {
  guardianSet?: PublicKey;
  config?: PublicKey;
  signatureSet: PublicKey;
  postedVaa?: PublicKey;
  payer: PublicKey;
  clock?: PublicKey;
  rent?: PublicKey;
};

export type LegacyPostVaaArgs = {
  // Version could have been passed as '1'. There is nothing from the signature set or other
  // post VAA arguments that can verify whether this version is correct.
  version?: number;
  // Another unnecessary argument because this guardian set index is already checked by the
  // Anchor account context (using the one encoded in the signature set). But because this
  // legacy instruction expects borsh serialization of enum VaaVersion, the only valid values
  // are 0 and 1.
  guardianSetIndex: number;
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: number[];
  sequence: BN;
  consistencyLevel: number;
  payload: Buffer;
};

export function legacyPostVaaIx(
  program: CoreBridgeProgram,
  accounts: LegacyPostVaaContext,
  args: LegacyPostVaaArgs
) {
  const programId = program.programId;
  const {
    version,
    guardianSetIndex,
    timestamp,
    nonce,
    emitterChain,
    emitterAddress,
    sequence,
    consistencyLevel,
    payload,
  } = args;

  let { guardianSet, config, signatureSet, postedVaa, payer, clock, rent } = accounts;

  if (guardianSet === undefined) {
    guardianSet = GuardianSet.address(programId, args.guardianSetIndex);
  }

  if (config === undefined) {
    config = Config.address(program.programId);
  }

  if (postedVaa === undefined) {
    // This is terrible, but we need to generate the message hash.
    const message = Buffer.alloc(51 + payload.length);
    message.writeUInt32BE(timestamp, 0);
    message.writeUInt32BE(nonce, 4);
    message.writeUInt16BE(emitterChain, 8);
    message.set(emitterAddress, 10);
    message.writeBigUInt64BE(BigInt(sequence.toString()), 42);
    message.writeUInt8(consistencyLevel, 50);
    message.set(payload, 51);

    postedVaa = PostedVaaV1.address(
      programId,
      Array.from(ethers.utils.arrayify(ethers.utils.keccak256(message)))
    );
  }

  if (clock === undefined) {
    clock = SYSVAR_CLOCK_PUBKEY;
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
  }

  const keys: AccountMeta[] = [
    {
      pubkey: guardianSet,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: config,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: signatureSet,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: postedVaa,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: clock,
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

  const data = Buffer.alloc(1 + 1 + 4 + 4 + 4 + 2 + 32 + 8 + 1 + 4 + payload.length);
  data.writeUInt8(2, 0);
  data.writeUInt8(version === undefined ? 0 : version, 1);
  data.writeUInt32LE(guardianSetIndex, 2);
  data.writeUInt32LE(timestamp, 6);
  data.writeUInt32LE(nonce, 10);
  data.writeUInt16LE(emitterChain, 14);
  data.set(emitterAddress, 16);
  data.writeBigUInt64LE(BigInt(sequence.toString()), 48);
  data.writeUInt8(consistencyLevel, 56);
  data.writeUInt32LE(payload.length, 57);
  data.set(payload, 61);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
