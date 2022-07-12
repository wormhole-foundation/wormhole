import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export function deriveCustodyKey(
  tokenBridgeProgramId: PublicKeyInitData,
  mint: PublicKeyInitData
): PublicKey {
  return deriveAddress([new PublicKey(mint).toBuffer()], tokenBridgeProgramId);
}
