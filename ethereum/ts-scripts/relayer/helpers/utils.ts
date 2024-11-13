import { ContractReceipt, ContractTransaction } from "ethers";
import { chainToPlatform, UniversalAddress, toChainId, chainIdToChain, platformToAddressFormat, ChainId } from "@wormhole-foundation/sdk";

export function wait(tx: ContractTransaction): Promise<ContractReceipt> {
  return tx.wait();
}

export function nativeEthereumAddressToHex(address: string): string {
  return nativeAddressToHex(address, toChainId("Ethereum"));
}

export function nativeAddressToHex(address: string, chainId: ChainId): string {
  const platform = chainToPlatform(chainIdToChain(chainId))
  return (new UniversalAddress(address, platformToAddressFormat(platform))).toString();
}
