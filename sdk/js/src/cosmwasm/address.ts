import { keccak256 } from "ethers/lib/utils";
import { canonicalAddress, isNativeDenom } from "../terra";

export function buildTokenId(address: string) {
  if (isNativeDenom(address)) {
    return "01" + keccak256(Buffer.from(address, "utf-8")).substring(4);
  } else {
    return "00" + keccak256(canonicalAddress(address)).substring(4);
  }
}
