import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils";

export function deriveWormholeBridgeDataKey(
  wormholeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("Bridge")], wormholeProgramId);
}

export async function getWormholeBridgeData(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  commitment?: Commitment
): Promise<BridgeData> {
  return connection
    .getAccountInfo(deriveWormholeBridgeDataKey(wormholeProgramId), commitment)
    .then((info) => BridgeData.deserialize(getAccountData(info)));
}

export class BridgeConfig {
  guardianSetExpirationTime: number;
  fee: bigint;

  constructor(guardianSetExpirationTime: number, fee: bigint) {
    this.guardianSetExpirationTime = guardianSetExpirationTime;
    this.fee = fee;
  }

  static deserialize(data: Buffer): BridgeConfig {
    if (data.length != 12) {
      throw new Error("data.length != 12");
    }
    const guardianSetExpirationTime = data.readUInt32LE(0);
    const fee = data.readBigUInt64LE(4);
    return new BridgeConfig(guardianSetExpirationTime, fee);
  }
}

export class BridgeData {
  guardianSetIndex: number;
  lastLamports: bigint;
  config: BridgeConfig;

  constructor(
    guardianSetIndex: number,
    lastLamports: bigint,
    config: BridgeConfig
  ) {
    this.guardianSetIndex = guardianSetIndex;
    this.lastLamports = lastLamports;
    this.config = config;
  }

  static deserialize(data: Buffer): BridgeData {
    if (data.length != 24) {
      throw new Error("data.length != 24");
    }
    const guardianSetIndex = data.readUInt32LE(0);
    const lastLamports = data.readBigUInt64LE(4);
    const config = BridgeConfig.deserialize(data.subarray(12));
    return new BridgeData(guardianSetIndex, lastLamports, config);
  }
}
