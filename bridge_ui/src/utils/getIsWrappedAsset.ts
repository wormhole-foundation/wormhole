import { getIsWrappedAssetSol as getIsWrappedAssetSolTx } from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { SOLANA_HOST, SOL_TOKEN_BRIDGE_ADDRESS } from "./consts";

export async function getIsWrappedAssetSol(mintAddress: string) {
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  return await getIsWrappedAssetSolTx(
    connection,
    SOL_TOKEN_BRIDGE_ADDRESS,
    mintAddress
  );
}
