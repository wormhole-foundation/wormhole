import { Connection, PublicKey, Commitment, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils/account";

export function bridgeInfoKey(wormholeProgramId: PublicKeyInitData): PublicKey {
  return deriveAddress([Buffer.from("Bridge")], wormholeProgramId);
}

export async function getBridgeInfo(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  commitment?: Commitment
): Promise<BridgeData> {
  return connection
    .getAccountInfo(bridgeInfoKey(wormholeProgramId), commitment)
    .then((info) => BridgeData.deserialize(getAccountData(info)));
}

export class BridgeConfig {
  // pub struct BridgeConfig {
  //     /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
  //     /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
  //     /// this period we still trust the old guardian set.
  //     pub guardian_set_expiration_time: u32,

  //     /// Amount of lamports that needs to be paid to the protocol to post a message
  //     pub fee: u64,
  // }

  guardianSetExpirationTime: number;
  fee: bigint;

  constructor(guardianSetExpirationTime: number, fee: bigint) {
    this.guardianSetExpirationTime = guardianSetExpirationTime;
    this.fee = fee;
  }

  static deserialize(data: Buffer): BridgeConfig {
    const guardianSetExpirationTime = data.readUInt32LE(0);
    const fee = data.readBigUInt64LE(4);
    return new BridgeConfig(guardianSetExpirationTime, fee);
  }
}

export class BridgeData {
  // pub struct BridgeData {
  //     /// The current guardian set index, used to decide which signature sets to accept.
  //     pub guardian_set_index: u32,

  //     /// Lamports in the collection account
  //     pub last_lamports: u64,

  //     /// Bridge configuration, which is set once upon initialization.
  //     pub config: BridgeConfig,
  // }

  guardianSetIndex: number;
  lastLamports: bigint;
  config: BridgeConfig;

  constructor(guardianSetIndex: number, lastLamports: bigint, config: BridgeConfig) {
    this.guardianSetIndex = guardianSetIndex;
    this.lastLamports = lastLamports;
    this.config = config;
  }

  static deserialize(data: Buffer): BridgeData {
    const guardianSetIndex = data.readUInt32LE(0);
    const lastLamports = data.readBigUInt64LE(4);
    const config = BridgeConfig.deserialize(data.subarray(12));
    return new BridgeData(guardianSetIndex, lastLamports, config);
  }
}
