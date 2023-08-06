import { BN } from "@coral-xyz/anchor";
import {
  AccountMeta,
  PublicKey,
  PublicKeyInitData,
  SYSVAR_CLOCK_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { ProgramId } from "../../consts";
import { BridgeProgramData, GuardianSet, PostedVaaV1 } from "../../state";
import { getProgramPubkey } from "../../utils/misc";

/* private */
export type PostVaaConfig = {
  bridge: boolean;
  clock: boolean;
  rent: boolean;
};

export class LegacyPostVaaContext {
  guardianSet: PublicKey;
  _bridge: PublicKey | null;
  signatureSet: PublicKey;
  postedVaa: PublicKey;
  payer: PublicKey;
  _clock: PublicKey | null;
  _rent: PublicKey | null;
  systemProgram: PublicKey;

  protected constructor(
    programId: ProgramId,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    hash: number[],
    payer: PublicKeyInitData,
    config: PostVaaConfig
  ) {
    this.guardianSet = GuardianSet.address(programId, guardianSetIndex);
    this._bridge = config.bridge ? BridgeProgramData.address(programId) : null;
    this.signatureSet = new PublicKey(signatureSet);
    this.postedVaa = PostedVaaV1.address(programId, hash);
    this.payer = new PublicKey(payer);
    this._clock = config.clock ? SYSVAR_CLOCK_PUBKEY : null;
    this._rent = config.rent ? SYSVAR_RENT_PUBKEY : null;
    this.systemProgram = SystemProgram.programId;
  }

  static new(
    programId: ProgramId,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    hash: number[],
    payer: PublicKeyInitData,
    config: PostVaaConfig
  ) {
    return new LegacyPostVaaContext(
      programId,
      guardianSetIndex,
      signatureSet,
      hash,
      payer,
      config
    );
  }

  static instruction(
    programId: ProgramId,
    guardianSetIndex: number,
    signatureSet: PublicKeyInitData,
    hash: number[],
    payer: PublicKeyInitData,
    args: LegacyPostVaaArgs,
    config?: PostVaaConfig
  ) {
    return legacyPostVaaIx(
      programId,
      LegacyPostVaaContext.new(
        programId,
        guardianSetIndex,
        signatureSet,
        hash,
        payer,
        config || { bridge: false, clock: false, rent: false }
      ),
      args
    );
  }
}

export type LegacyPostVaaArgs = {
  // Version could have been passed as '1'. There is nothing from the signature set or other
  // post VAA arguments that can verify whether this version is correct.
  version?: number;
  // Another unnecessary argument because this guardian set index is already checked by the
  // Anchor account context (using the one encoded in the signature set). But because this
  // legacy instruction expects borsh serialization of enum VaaVersion, the only valid values
  // are 0 and 1.
  guardianSetIndex?: number;
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: number[];
  sequence: BN;
  finality: number;
  payload: Buffer;
};

export function legacyPostVaaIx(
  programId: ProgramId,
  accounts: LegacyPostVaaContext,
  args: LegacyPostVaaArgs
) {
  const thisProgramId = getProgramPubkey(programId);
  const {
    guardianSet,
    _bridge,
    signatureSet,
    postedVaa,
    payer,
    _clock,
    _rent,
    systemProgram,
  } = accounts;
  const keys: AccountMeta[] = [
    {
      pubkey: guardianSet,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: _bridge === null ? thisProgramId : _bridge,
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
      pubkey: _clock === null ? thisProgramId : _clock,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: _rent === null ? thisProgramId : _rent,
      isWritable: false,
      isSigner: false,
    },
    {
      pubkey: systemProgram,
      isWritable: false,
      isSigner: false,
    },
  ];
  const {
    version,
    guardianSetIndex,
    timestamp,
    nonce,
    emitterChain,
    emitterAddress,
    sequence,
    finality,
    payload,
  } = args;
  const data = Buffer.alloc(
    1 + 1 + 4 + 4 + 4 + 2 + 32 + 8 + 1 + 4 + payload.length
  );
  data.writeUInt8(2, 0);
  data.writeUInt8(version === undefined ? 0 : version, 1);
  data.writeUInt32LE(guardianSetIndex === undefined ? 0 : guardianSetIndex, 2);
  data.writeUInt32LE(timestamp, 6);
  data.writeUInt32LE(nonce, 10);
  data.writeUInt16LE(emitterChain, 14);
  data.write(Buffer.from(emitterAddress).toString("hex"), 16, "hex");
  data.writeBigUInt64LE(BigInt(sequence.toString()), 48);
  data.writeUInt8(finality, 56);
  data.writeUInt32LE(payload.length, 57);
  data.write(payload.toString("hex"), 61, "hex");

  return new TransactionInstruction({
    keys,
    programId: thisProgramId,
    data,
  });
}
