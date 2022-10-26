import { PublicKeyInitData } from "@solana/web3.js";
import { deriveClaimKey } from "../solana/wormhole";
import { parseVaa, SignedVaa } from "../vaa/wormhole";

export async function getClaimAddressSolana(
  programAddress: PublicKeyInitData,
  signedVaa: SignedVaa
) {
  const parsed = parseVaa(signedVaa);
  return deriveClaimKey(
    programAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence
  );
}
