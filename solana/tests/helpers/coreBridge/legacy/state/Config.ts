import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export class Config {
  guardianSetIndex: number;
  lastLamports: BN;
  guardianSetTtl: number;
  feeLamports: BN;

  private constructor(
    guardianSetIndex: number,
    lastLamports: BN,
    guardianSetTtl: number,
    feeLamports: BN
  ) {
    this.guardianSetIndex = guardianSetIndex;
    this.lastLamports = lastLamports;
    this.guardianSetTtl = guardianSetTtl;
    this.feeLamports = feeLamports;
  }

  static address(programId: PublicKey): PublicKey {
    return PublicKey.findProgramAddressSync([Buffer.from("Bridge")], programId)[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): Config {
    return Config.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<Config> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find Config account at ${address}`);
    }
    return Config.fromAccountInfo(accountInfo);
  }

  static async fromPda(connection: Connection, programId: PublicKey): Promise<Config> {
    return Config.fromAccountAddress(connection, Config.address(programId));
  }

  static deserialize(data: Buffer): Config {
    if (data.length != 24) {
      throw new Error("data.length != 24");
    }
    const guardianSetIndex = data.readUInt32LE(0);
    const lastLamports = new BN(data.subarray(4, 12), undefined, "le");

    const guardianSetExpirationTime = data.readUInt32LE(12);
    const feeLamports = new BN(data.subarray(16, 24), undefined, "le");
    return new Config(guardianSetIndex, lastLamports, guardianSetExpirationTime, feeLamports);
  }
}
