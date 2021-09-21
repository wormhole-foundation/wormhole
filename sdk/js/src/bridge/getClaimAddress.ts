import { PublicKey } from "@solana/web3.js";

export async function getClaimAddressSolana(
  programAddress: string,
  signedVAA: Uint8Array
) {
  const { claim_address } = await import("../solana/core/bridge");
  return new PublicKey(claim_address(programAddress, signedVAA));
}
