import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";
import { ProgramId } from "../consts";
import { getProgramPubkey } from "../utils";

export class BridgeProgramData {
  guardianSetIndex: number;
  lastLamports: BN;
  config: BridgeConfig;

  private constructor(
    guardianSetIndex: number,
    lastLamports: BN,
    config: BridgeConfig
  ) {
    this.guardianSetIndex = guardianSetIndex;
    this.lastLamports = lastLamports;
    this.config = config;
  }

  static address(programId: ProgramId): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("Bridge")],
      getProgramPubkey(programId)
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): BridgeProgramData {
    return BridgeProgramData.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<BridgeProgramData> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find BridgeProgramData account at ${address}`);
    }
    return BridgeProgramData.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: ProgramId
  ): Promise<BridgeProgramData> {
    return BridgeProgramData.fromAccountAddress(
      connection,
      BridgeProgramData.address(programId)
    );
  }

  static deserialize(data: Buffer): BridgeProgramData {
    if (data.length != 24) {
      throw new Error("data.length != 24");
    }
    const guardianSetIndex = data.readUInt32LE(0);
    const lastLamports = new BN(data.subarray(4, 12), undefined, "le");
    const config = BridgeConfig.deserialize(data.subarray(12));
    return new BridgeProgramData(guardianSetIndex, lastLamports, config);
  }
}

export class BridgeConfig {
  guardianSetTtl: number;
  feeLamports: BN;

  private constructor(guardianSetTtl: number, feeLamports: BN) {
    this.guardianSetTtl = guardianSetTtl;
    this.feeLamports = feeLamports;
  }

  static deserialize(data: Buffer): BridgeConfig {
    if (data.length != 12) {
      throw new Error("data.length != 12");
    }
    const guardianSetExpirationTime = data.readUInt32LE(0);
    const feeLamports = new BN(data.subarray(4, 12), undefined, "le");
    return new BridgeConfig(guardianSetExpirationTime, feeLamports);
  }
}
