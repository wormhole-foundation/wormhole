import { keccak256 } from "../utils";
import {parseVaaV1, SignedVaa} from "../vaa/wormhole";

export function getSignedVAAHash(signedVaa: SignedVaa): string {
  return `0x${keccak256(parseVaaV1(signedVaa).hash).toString("hex")}`;
}
