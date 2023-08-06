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
import { Config, GuardianPubkey, GuardianSet, feeCollectorPda } from "../state";

export type LegacyInitializeContext = {
  bridge?: PublicKey;
  guardianSet?: PublicKey;
  feeCollector?: PublicKey;
  payer: PublicKey;
};

export type LegacyInitializeArgs = {
  guardianSetTtlSeconds: number;
  feeLamports: BN;
  initialGuardians: GuardianPubkey[];
};

export function legacyInitializeIx(
  program: CoreBridgeProgram,
  accounts: LegacyInitializeContext,
  args: LegacyInitializeArgs
) {
  const programId = program.programId;

  let { bridge, guardianSet, feeCollector, payer } = accounts;

  if (bridge === undefined) {
    bridge = Config.address(programId);
  }

  if (guardianSet === undefined) {
    guardianSet = GuardianSet.address(programId, 0);
  }

  if (feeCollector === undefined) {
    feeCollector = feeCollectorPda(programId);
  }

  const keys: AccountMeta[] = [
    {
      pubkey: bridge,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: guardianSet,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: feeCollector,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: payer,
      isWritable: false,
      isSigner: true,
    },
    {
      pubkey: SYSVAR_CLOCK_PUBKEY,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SYSVAR_RENT_PUBKEY,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: SystemProgram.programId,
      isWritable: false,
      isSigner: false,
    },
  ];

  const { guardianSetTtlSeconds, feeLamports, initialGuardians } = args;
  const data = Buffer.alloc(1 + 4 + 8 + 4 + 20 * initialGuardians.length);
  data.writeUInt8(0, 0);
  data.writeUInt32LE(guardianSetTtlSeconds, 1);
  data.writeBigInt64LE(BigInt(feeLamports.toString()), 5);

  const numGuardians = initialGuardians.length;
  data.writeUInt32LE(numGuardians, 13);
  for (let i = 0; i < numGuardians; ++i) {
    const guardian = initialGuardians[i];
    data.set(guardian, 1 + 4 + 8 + 4 + i * 20);
  }

  return new TransactionInstruction({
    keys,
    programId,
    data,
  });
}
