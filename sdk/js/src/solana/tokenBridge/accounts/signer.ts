import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export function deriveAuthoritySignerKey(
  tokenBridgeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("authority_signer")], tokenBridgeProgramId);
}

export function deriveCustodySignerKey(
  tokenBridgeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("custody_signer")], tokenBridgeProgramId);
}

export function deriveMintAuthorityKey(
  tokenBridgeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("mint_signer")], tokenBridgeProgramId);
}
