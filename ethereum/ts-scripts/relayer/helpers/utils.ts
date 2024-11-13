import { ContractReceipt, ContractTransaction } from "ethers";
import * as wh from "@wormhole-foundation/sdk";

export function wait(tx: ContractTransaction): Promise<ContractReceipt> {
  return tx.wait();
}

export function nativeEvmAddressToHex(address: string): string {
  return (new wh.UniversalAddress(address, wh.platformToAddressFormat("Evm"))).toString();
}
