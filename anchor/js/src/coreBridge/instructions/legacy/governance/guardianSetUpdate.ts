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

export class LegacyGuardianSetUpdateContext {
  payer: PublicKey;
  bridge: PublicKey;
  postedVaa: PublicKey;
  claim: PublicKey;
  currGuardianSet: PublicKey;
  newGuardianSet: PublicKey;
  systemProgram: PublicKey;

  protected constructor(
    programId: ProgramId,
    payer: PublicKeyInitData,
    currGuardianSet: PublicKeyInitData,
    newGuardianSet: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    this.payer = new PublicKey(payer);
    this.bridge = BridgeProgramData.address(programId);
    this.postedVaa = PostedVaaV1.address(programId, hash);
    this.claim = Claim.address(getProgramPubkey(programId), vaaInfo);
    this.currGuardianSet = new PublicKey(currGuardianSet);
    this.newGuardianSet = new PublicKey(newGuardianSet);
    this.systemProgram = SystemProgram.programId;
  }

  static new(
    programId: ProgramId,
    payer: PublicKeyInitData,
    currGuardianSet: PublicKeyInitData,
    newGuardianSet: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    return new LegacyGuardianSetUpdateContext(
      programId,
      payer,
      currGuardianSet,
      newGuardianSet,
      hash,
      vaaInfo
    );
  }

  static instruction(
    programId: ProgramId,
    payer: PublicKeyInitData,
    currGuardianSet: PublicKeyInitData,
    newGuardianSet: PublicKeyInitData,
    hash: number[],
    vaaInfo: VaaInfo
  ) {
    return legacyGuardianSetUpdateIx(
      programId,
      LegacyGuardianSetUpdateContext.new(
        programId,
        payer,
        currGuardianSet,
        newGuardianSet,
        hash,
        vaaInfo
      )
    );
  }
}

export function legacyGuardianSetUpdateIx(
  programId: ProgramId,
  accounts: LegacyGuardianSetUpdateContext
) {
  const {
    payer,
    bridge,
    postedVaa,
    claim,
    currGuardianSet,
    newGuardianSet,
    systemProgram,
  } = accounts;
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
      pubkey: currGuardianSet,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: newGuardianSet,
      isWritable: true,
      isSigner: false,
    },
    {
      pubkey: systemProgram,
      isWritable: false,
      isSigner: false,
    },
  ];
  const data = Buffer.alloc(1, 6);

  return new TransactionInstruction({
    keys,
    programId: getProgramPubkey(programId),
    data,
  });
}
