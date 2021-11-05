import { PublicKey } from "@solana/web3.js";
import { importCoreWasm } from "../solana/wasm";

export async function getClaimAddressSolana(
  programAddress: string,
  signedVAA: Uint8Array
) {
  const { claim_address } = await importCoreWasm();
  return new PublicKey(claim_address(programAddress, signedVAA));
}
