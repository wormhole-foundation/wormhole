import { TOKEN_PROGRAM_ID, getAssociatedTokenAddressSync } from "@solana/spl-token";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { LegacyTransferTokensWithPayloadArgs } from "..";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import * as coreBridge from "../../../../coreBridge";
import { Config, WrappedAsset, coreEmitterPda, transferAuthorityPda } from "../../state";

export type LegacyTransferTokensWithPayloadWrappedContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  srcToken?: PublicKey;
  srcOwner?: PublicKey;
  wrappedMint: PublicKey;
  wrappedAsset?: PublicKey;
  transferAuthority?: PublicKey;
  coreBridgeConfig?: PublicKey;
  coreMessage: PublicKey;
  coreEmitter?: PublicKey;
  coreEmitterSequence?: PublicKey;
  coreFeeCollector?: PublicKey;
  clock?: PublicKey; // TODO: demonstrate this isn't needed in tests
  senderAuthority?: PublicKey;
  rent?: PublicKey; // TODO: demonstrate this isn't needed in tests
  coreBridgeProgram?: PublicKey;
};

export function legacyTransferTokensWithPayloadWrappedAccounts(
  program: TokenBridgeProgram,
  accounts: LegacyTransferTokensWithPayloadWrappedContext
): LegacyTransferTokensWithPayloadWrappedContext {
  const programId = program.programId;

  let {
    payer,
    config,
    srcToken,
    srcOwner,
    wrappedMint,
    wrappedAsset,
    transferAuthority,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    senderAuthority,
    rent,
    coreBridgeProgram,
  } = accounts;

  if (coreBridgeProgram === undefined) {
    coreBridgeProgram = coreBridgeProgramId(program);
  }

  if (config === undefined) {
    config = Config.address(programId);
  }

  if (srcToken === undefined) {
    srcToken = getAssociatedTokenAddressSync(wrappedMint, payer);
  }

  if (srcOwner === undefined) {
    srcOwner = payer;
  }

  if (wrappedAsset === undefined) {
    wrappedAsset = WrappedAsset.address(programId, wrappedMint);
  }

  if (transferAuthority === undefined) {
    transferAuthority = transferAuthorityPda(programId);
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

  if (senderAuthority === undefined) {
    senderAuthority = payer;
  }

  if (clock === undefined) {
    clock = SYSVAR_CLOCK_PUBKEY;
  }

  if (rent === undefined) {
    rent = SYSVAR_RENT_PUBKEY;
  }

  return {
    payer,
    config,
    srcToken,
    srcOwner,
    wrappedMint,
    wrappedAsset,
    transferAuthority,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    senderAuthority,
    rent,
    coreBridgeProgram,
  };
}

export function legacyTransferTokensWithPayloadWrappedIx(
  program: TokenBridgeProgram,
  accounts: LegacyTransferTokensWithPayloadWrappedContext,
  args: LegacyTransferTokensWithPayloadArgs
) {
  const {
    payer,
    config,
    srcToken,
    srcOwner,
    wrappedMint,
    wrappedAsset,
    transferAuthority,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    senderAuthority,
    rent,
    coreBridgeProgram,
  } = legacyTransferTokensWithPayloadWrappedAccounts(program, accounts);

  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: config!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: srcToken!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: srcOwner!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: wrappedMint,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: wrappedAsset!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: transferAuthority!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreBridgeConfig!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: coreMessage,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: coreEmitter!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreEmitterSequence!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: coreFeeCollector!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: clock!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: senderAuthority!,
      isWritable: false,
      isSigner: true,
    },
    {
      pubkey: rent!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: TOKEN_PROGRAM_ID,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: coreBridgeProgram!,
      isWritable: false,
      isSigner: false,
    },
  ];

  const { nonce, amount, redeemer, redeemerChain, payload, cpiProgramId } = args;
  const cpiRequired = cpiProgramId !== null;
  const data = Buffer.alloc(1 + 4 + 8 + 32 + 2 + 4 + payload.length + 1 + (cpiRequired ? 32 : 0));
  data.writeUInt8(11, 0);
  data.writeUInt32LE(nonce, 1);
  data.writeBigUInt64LE(BigInt(amount.toString()), 5);
  data.set(redeemer, 13);
  data.writeUInt16LE(redeemerChain, 45);
  data.writeUInt32LE(payload.length, 47);
  data.set(payload, 51);
  if (cpiRequired) {
    data.writeUInt8(1, 51 + payload.length);
    data.set(cpiProgramId.toBuffer(), 52 + payload.length);
  }

  return new TransactionInstruction({
    keys,
    programId: program.programId,
    data,
  });
}
