import { PublicKey } from "@solana/web3.js";
import { arrayify, BytesLike, Hexable, zeroPad } from "ethers/lib/utils";

export function getEmitterAddressEth(
  contractAddress: number | BytesLike | Hexable
) {
  return Buffer.from(zeroPad(arrayify(contractAddress), 32)).toString("hex");
}

export async function getEmitterAddressSolana(programAddress: string) {
  const { emitter_address } = await import("../solana/token/token_bridge");
  return Buffer.from(
    zeroPad(new PublicKey(emitter_address(programAddress)).toBytes(), 32)
  ).toString("hex");
}

export async function getEmitterAddressTerra(programAddress: string) {
  // Testnet Hardcoded
  // TODO: HumanAddr -> CanonicalAddr
  return "000000000000000000000000784999135aaa8a3ca5914468852fdddbddd8789d";
}
