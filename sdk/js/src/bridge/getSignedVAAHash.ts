import { keccak256 } from "../utils";
import { parseVaa, SignedVaa } from "../vaa/wormhole";

export function getSignedVAAHash(signedVaa: SignedVaa): string {
  return `0x${keccak256(parseVaa(signedVaa).hash).toString("hex")}`;
}
