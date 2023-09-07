import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountMeta,
  PublicKey,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { LegacyTransferTokensArgs } from "../";
import { TokenBridgeProgram, coreBridgeProgramId } from "../../..";
import * as coreBridge from "../../../../coreBridge";
import {
  Config,
  coreEmitterPda,
  custodyAuthorityPda,
  custodyTokenPda,
  transferAuthorityPda,
} from "../../state";

export type LegacyTransferTokensNativeContext = {
  payer: PublicKey;
  config?: PublicKey; // TODO: demonstrate this isn't needed in tests
  srcToken: PublicKey;
  mint: PublicKey;
  custodyToken?: PublicKey;
  transferAuthority?: PublicKey;
  custodyAuthority?: PublicKey;
  coreBridgeConfig?: PublicKey;
  coreMessage: PublicKey;
  coreEmitter?: PublicKey;
  coreEmitterSequence?: PublicKey;
  coreFeeCollector?: PublicKey;
  clock?: PublicKey; // TODO: demonstrate this isn't needed in tests
  rent?: PublicKey; // TODO: demonstrate this isn't needed in tests
  coreBridgeProgram?: PublicKey;
};

export function legacyTransferTokensNativeAccounts(
  program: TokenBridgeProgram,
  accounts: LegacyTransferTokensNativeContext
): LegacyTransferTokensNativeContext {
  const programId = program.programId;

  let {
    payer,
    config,
    srcToken,
    mint,
    custodyToken,
    transferAuthority,
    custodyAuthority,
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

  if (custodyToken === undefined) {
    custodyToken = custodyTokenPda(programId, mint);
  }

  if (transferAuthority === undefined) {
    transferAuthority = transferAuthorityPda(programId);
  }

  if (custodyAuthority === undefined) {
    custodyAuthority = custodyAuthorityPda(programId);
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

  return {
    payer,
    config,
    srcToken,
    mint,
    custodyToken,
    transferAuthority,
    custodyAuthority,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    rent,
    coreBridgeProgram,
  };
}

export function legacyTransferTokensNativeIx(
  program: TokenBridgeProgram,
  accounts: LegacyTransferTokensNativeContext,
  args: LegacyTransferTokensArgs
) {
  const {
    payer,
    config,
    srcToken,
    mint,
    custodyToken,
    transferAuthority,
    custodyAuthority,
    coreBridgeConfig,
    coreMessage,
    coreEmitter,
    coreEmitterSequence,
    coreFeeCollector,
    clock,
    rent,
    coreBridgeProgram,
  } = legacyTransferTokensNativeAccounts(program, accounts);

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
      pubkey: srcToken,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: mint,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: custodyToken!,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: transferAuthority!,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: custodyAuthority!,
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

  const { nonce, amount, relayerFee, recipient, recipientChain } = args;
  const data = Buffer.alloc(1 + 4 + 8 + 8 + 32 + 2);
  data.writeUInt8(5, 0);
  data.writeUInt32LE(nonce, 1);
  data.writeBigUInt64LE(BigInt(amount.toString()), 5);
  data.writeBigUInt64LE(BigInt(relayerFee.toString()), 13);
  data.set(recipient, 21);
  data.writeUInt16LE(recipientChain, 53);

  return new TransactionInstruction({
    keys,
    programId: program.programId,
    data,
  });
}
