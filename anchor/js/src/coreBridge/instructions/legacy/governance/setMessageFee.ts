import {
  AccountMeta,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { ProgramId } from "../../../consts";
import { BridgeProgramData, Claim, PostedVaaV1, VaaInfo } from "../../../state";
import { getProgramPubkey } from "../../../utils/misc";

export class LegacySetMessageFeeContext {
  payer: PublicKey;
  bridge: PublicKey;
  postedVaa: PublicKey;
  claim: PublicKey;
  systemProgram: PublicKey;

  protected constructor(
    programId: ProgramId,
    payer: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    this.payer = new PublicKey(payer);
    this.bridge = BridgeProgramData.address(programId);
    this.postedVaa = PostedVaaV1.address(programId, hash);
    this.claim = Claim.address(getProgramPubkey(programId), vaaInfo);
    this.systemProgram = SystemProgram.programId;
  }

  static new(
    programId: ProgramId,
    payer: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    return new LegacySetMessageFeeContext(programId, payer, hash, vaaInfo);
  }

  static instruction(
    programId: ProgramId,
    payer: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    return legacySetMessageFeeIx(
      programId,
      LegacySetMessageFeeContext.new(programId, payer, hash, vaaInfo)
    );
  }
}

export function legacySetMessageFeeIx(
  programId: ProgramId,
  accounts: LegacySetMessageFeeContext
) {
  const { payer, bridge, postedVaa, claim, systemProgram } = accounts;
  const keys: AccountMeta[] = [
    {
      pubkey: payer,
      isWritable: true,
      isSigner: true,
    },
    {
      pubkey: bridge,
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
      pubkey: systemProgram,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 3);

  return new TransactionInstruction({
    keys,
    programId: getProgramPubkey(programId),
    data,
  });
}
