import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";
import { ProgramId } from "../consts";
import { getProgramPubkey } from "../utils";

export class Config {
  coreBridge: PublicKey;

  private constructor(coreBridge: PublicKey) {
    this.coreBridge = coreBridge;
  }

  static address(programId: ProgramId): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("config")],
      getProgramPubkey(programId)
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): Config {
    return Config.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<Config> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find BridgeProgramData account at ${address}`);
    }
    return Config.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: ProgramId
  ): Promise<Config> {
    return Config.fromAccountAddress(connection, Config.address(programId));
  }

  static deserialize(data: Buffer): Config {
    if (data.length != 32) {
      throw new Error("data.length != 32");
    }
    const coreBridge = new PublicKey(data);
    return new Config(coreBridge);
  }
}
