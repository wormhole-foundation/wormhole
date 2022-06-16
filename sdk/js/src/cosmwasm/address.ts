import { keccak256 } from "ethers/lib/utils";
import { isNativeDenom } from "../terra";

export function buildTokenId(address: string) {
  return (
    (isNativeDenom(address) ? "01" : "00") +
    keccak256(Buffer.from(address, "utf-8")).substring(4)
  );
}
