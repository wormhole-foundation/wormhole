import { PublicKeyInitData } from "@solana/web3.js";
import { claimKey } from "../solana";
import { parseVaa, SignedVaa } from "../vaa/wormhole";

export async function getClaimAddressSolana(
  programAddress: PublicKeyInitData,
  signedVaa: SignedVaa
) {
  const parsed = parseVaa(signedVaa);
  return claimKey(
    programAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence
  );
}
