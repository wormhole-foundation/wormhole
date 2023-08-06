import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { TokenBridgeProgram, coreBridgeProgramId } from "../..";
import { Config, coreEmitterPda, tokenMetadataPda, WrappedAsset } from "../state";
import * as coreBridge from "../../../coreBridge";

export type LegacyAttestTokenContext = {
  payer: PublicKey;
  config?: PublicKey;
  mint: PublicKey;
  nativeAsset?: PublicKey; // TODO: demonstrate this isn't needed in tests
  tokenMetadata?: PublicKey;
  coreBridgeConfig?: PublicKey;
  coreMessage: PublicKey;
  coreEmitter?: PublicKey;
  coreEmitterSequence?: PublicKey;
  coreFeeCollector?: PublicKey;
  clock?: PublicKey; // TODO: demonstrate this isn't needed in tests
  rent?: PublicKey; // TODO: demonstrate this isn't needed in tests
  coreBridgeProgram?: PublicKey;
};

export type LegacyAttestTokenArgs = {
  nonce: number;
};

export function legacyAttestTokenIx(
  program: TokenBridgeProgram,
  accounts: LegacyAttestTokenContext,
  args: LegacyAttestTokenArgs
) {
  const programId = program.programId;

  let {
    payer,
    config,
    mint,
    nativeAsset,
    tokenMetadata,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    rent,
    coreBridgeProgram,
  } = accounts;

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (nativeAsset === undefined) {
    nativeAsset = WrappedAsset.address(programId, mint);
  }

  if (tokenMetadata === undefined) {
    tokenMetadata = tokenMetadataPda(mint);
  }

  if (coreBridgeConfig === undefined) {
    coreBridgeConfig = coreBridge.Config.address(coreBridgeProgram);
  }

  if (coreEmitter === undefined) {
    coreEmitter = coreEmitterPda(programId);
  }

  if (coreEmitterSequence === undefined) {
    coreEmitterSequence = coreBridge.EmitterSequence.address(coreBridgeProgram, coreEmitter);
  }

  if (coreFeeCollector === undefined) {
    coreFeeCollector = coreBridge.feeCollectorPda(coreBridgeProgram);
  }

  if (clock === undefined) {
    clock = SYSVAR_CLOCK_PUBKEY;
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
      isWritable: true, // bug in the program
      isSigner: false,
    },
    {
      pubkey: mint,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: nativeAsset,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: tokenMetadata,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreBridgeConfig,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: coreMessage,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: coreEmitter,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreEmitterSequence,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: coreFeeCollector,
      isWritable: true,
      isSigner: false,
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
    {
      pubkey: coreBridgeProgram,
      isWritable: false,
      isSigner: false,
    },
  ];

  const { nonce } = args;
  const data = Buffer.alloc(1 + 4);
  data.writeUInt8(1, 0);
  data.writeUInt32LE(nonce, 1);

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
