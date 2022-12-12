import { PublicKeyInitData } from "@solana/web3.js";
import { deriveClaimKey } from "../solana/wormhole";
import {parseVaaV1, SignedVaa} from "../vaa/wormhole";

export async function getClaimAddressSolana(
  programAddress: PublicKeyInitData,
  signedVaa: SignedVaa
) {
  const parsed = parseVaaV1(signedVaa);
  return deriveClaimKey(
    programAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence
  );
}
