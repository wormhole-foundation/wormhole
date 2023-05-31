import {
  AccountMeta,
  PublicKey,
  PublicKeyInitData,
  SYSVAR_INSTRUCTIONS_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { ProgramId } from "../../consts";
import { GuardianSet } from "../../state";
import { getProgramPubkey } from "../../utils/misc";

/* private */
export type VerifySignaturesConfig = {
  rent: boolean;
};

export class LegacyVerifySignaturesContext {
  payer: PublicKey;
  guardianSet: PublicKey;
  signatureSet: PublicKey;
  instructions: PublicKey;
  _rent: PublicKey;
  systemProgram: PublicKey;

  protected constructor(
    programId: ProgramId,
    payer: PublicKeyInitData,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    config: VerifySignaturesConfig
  ) {
    this.payer = new PublicKey(payer);
    this.guardianSet = GuardianSet.address(programId, guardianSetIndex);
    this.signatureSet = new PublicKey(signatureSet);
    this.instructions = SYSVAR_INSTRUCTIONS_PUBKEY;
    this._rent = config.rent ? SYSVAR_RENT_PUBKEY : SystemProgram.programId;
    this.systemProgram = SystemProgram.programId;
  }

  static new(
    programId: ProgramId,
    payer: PublicKeyInitData,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    config: VerifySignaturesConfig
  ) {
    return new LegacyVerifySignaturesContext(
      programId,
      payer,
      guardianSetIndex,
      signatureSet,
      config
    );
  }

  static instruction(
    programId: ProgramId,
    payer: PublicKeyInitData,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    args: LegacyVerifySignaturesArgs,
    config?: VerifySignaturesConfig
  ) {
    return legacyVerifySignaturesIx(
      programId,
      LegacyVerifySignaturesContext.new(
        programId,
        payer,
        guardianSetIndex,
        signatureSet,
        config || { rent: false }
      ),
      args
    );
  }
}

export type LegacyVerifySignaturesArgs = {
  signerIndices: number[];
};

export function legacyVerifySignaturesIx(
  programId: ProgramId,
  accounts: LegacyVerifySignaturesContext,
  args: LegacyVerifySignaturesArgs
) {
  const {
    payer,
    guardianSet,
    signatureSet,
    instructions,
    _rent,
    systemProgram,
  } = accounts;
  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: guardianSet,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: signatureSet,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: instructions,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: _rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: systemProgram,
      isWritable: false,
      isSigner: false,
    },
  ];
  const { signerIndices } = args;
  const numSigners = signerIndices.length;
  const data = Buffer.alloc(1 + numSigners);
  data.writeUInt8(7, 0);
  for (let i = 0; i < numSigners; ++i) {
    data.writeInt8(signerIndices[i], i + 1);
  }

  return new TransactionInstruction({
    keys,
    programId: getProgramPubkey(programId),
    data,
  });
}
