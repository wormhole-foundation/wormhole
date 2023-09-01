import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export class PostedVaaV1 {
  consistencyLevel: number;
  timestamp: number;
  signatureSet: PublicKey;
  guardianSetIndex: number;
  nonce: number;
  sequence: BN;
  emitterChain: number;
  emitterAddress: number[];
  payload: Buffer;

  private constructor(
    consistencyLevel: number,
    timestamp: number,
    signatureSet: PublicKey,
    guardianSetIndex: number,
    nonce: number,
    sequence: BN,
    emitterChain: number,
    emitterAddress: number[],
    payload: Buffer
  ) {
    this.consistencyLevel = consistencyLevel;
    this.timestamp = timestamp;
    this.signatureSet = signatureSet;
    this.guardianSetIndex = guardianSetIndex;
    this.nonce = nonce;
    this.sequence = sequence;
    this.emitterChain = emitterChain;
    this.emitterAddress = emitterAddress;
    this.payload = payload;
  }

  static address(programId: PublicKey, hash: number[]): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("PostedVAA"), Buffer.from(hash)],
      programId
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): PostedVaaV1 {
    const data = info.data;
    const discriminator = data.subarray(0, 4);
    if (!discriminator.equals(Buffer.from([118, 97, 97, 1]))) {
      throw new Error(`Invalid discriminator: ${discriminator}`);
    }
    return PostedVaaV1.deserialize(data.subarray(4));
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<PostedVaaV1> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find PostedVaaV1 account at ${address}`);
    }
    return PostedVaaV1.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: PublicKey,
    hash: number[]
  ): Promise<PostedVaaV1> {
    return PostedVaaV1.fromAccountAddress(connection, PostedVaaV1.address(programId, hash));
  }

  static deserialize(data: Buffer): PostedVaaV1 {
    const consistencyLevel = data.readUInt8(0);
    const timestamp = data.readUInt32LE(1);
    const signatureSet = new PublicKey(data.subarray(5, 37));
    const guardianSetIndex = data.readUInt32LE(37);
    const nonce = data.readUInt32LE(41);
    const sequence = new BN(data.subarray(45, 53), "le");
    const emitterChain = data.readUInt16LE(53);
    const emitterAddress = Array.from(data.subarray(55, 87));
    const payloadLen = data.readUInt32LE(87);
    const payload = data.subarray(91, 91 + payloadLen);

    return new PostedVaaV1(
      consistencyLevel,
      timestamp,
      signatureSet,
      guardianSetIndex,
      nonce,
      sequence,
      emitterChain,
      emitterAddress,
      payload
    );
  }
}
