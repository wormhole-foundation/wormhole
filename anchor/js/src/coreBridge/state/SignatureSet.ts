import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export class SignatureSet {
  sigVerifySuccesses: boolean[];
  messageHash: number[];
  guardianSetIndex: number;

  private constructor(
    sigVerifySiccesses: boolean[],
    messageHash: number[],
    guardianSetIndex: number
  ) {
    this.sigVerifySuccesses = sigVerifySiccesses;
    this.messageHash = messageHash;
    this.guardianSetIndex = guardianSetIndex;
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): SignatureSet {
    return SignatureSet.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<SignatureSet> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find SignatureSet account at ${address}`);
    }
    return SignatureSet.fromAccountInfo(accountInfo);
  }

  static deserialize(data: Buffer): SignatureSet {
    const numVerified = data.readUInt32LE(0);
    if (data.length != 40 + numVerified) {
      throw new Error("Invalid SignatureSet length");
    }
    const sigVerifySuccesses = Array.from(
      data.subarray(4, 4 + numVerified)
    ).map((value) => value != 0);
    const messageHash = Array.from(
      data.subarray(4 + numVerified, 36 + numVerified)
    );
    const guardianSetIndex = data.readUInt32LE(36 + numVerified);
    return new SignatureSet(sigVerifySuccesses, messageHash, guardianSetIndex);
  }
}
