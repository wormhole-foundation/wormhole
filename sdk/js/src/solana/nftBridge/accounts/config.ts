import { Connection, Commitment, PublicKeyInitData } from "@solana/web3.js";
import {
  deriveTokenBridgeConfigKey,
  getTokenBridgeConfig,
  TokenBridgeConfig,
} from "../../tokenBridge";

export const deriveNftBridgeConfigKey = deriveTokenBridgeConfigKey;

export async function getNftBridgeConfig(
  connection: Connection,
  nftBridgeProgramId: PublicKeyInitData,
  commitment?: Commitment
): Promise<NftBridgeConfig> {
  return getTokenBridgeConfig(connection, nftBridgeProgramId, commitment);
}

export class NftBridgeConfig extends TokenBridgeConfig {}
