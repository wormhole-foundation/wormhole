import { encoding } from "../../utils";
import { ChainName } from "../chains";
import { Network } from "../networks";
import { PlatformName, PlatformToChains, chainToPlatform } from "../platforms";
import { algorandChainIdToNetworkChain, algorandNetworkChainToChainId } from "./algorand";
import { aptosChainIdToNetworkChain, aptosNetworkChainToChainId } from "./aptos";
import { cosmwasmChainIdToNetworkChainPair, cosmwasmNetworkChainToChainId } from "./cosmwasm";
import { evmChainIdToNetworkChainPair, evmNetworkChainToEvmChainId } from "./evm";
import { nearChainIdToNetworkChain, nearNetworkChainToChainId } from "./near";
import { solGenesisHashToNetworkChainPair, solNetworkChainToGenesisHash } from "./solana";
import { suiChainIdToNetworkChain, suiNetworkChainToChainId } from "./sui";

export function getNativeChainId(n: Network, c: ChainName): string {
  const platform = chainToPlatform(c) as PlatformName;
  switch (platform) {
    case "Evm":
      return evmNetworkChainToEvmChainId.get(n, c).toString();
    case "Cosmwasm":
      return cosmwasmNetworkChainToChainId.get(n, c);
    case "Solana":
      return solNetworkChainToGenesisHash.get(n, c);
    case "Sui":
      return suiNetworkChainToChainId.get(n, c);
    case "Near":
      return nearNetworkChainToChainId.get(n, c);
    case "Aptos":
      return aptosNetworkChainToChainId.get(n, c).toString();
    case "Algorand":
      return algorandNetworkChainToChainId.get(n, c);
    case "Btc":
      throw new Error("unmapped");
  }
}

export function getNetworkAndChainName<P extends PlatformName>(
  platform: P,
  chainId: string,
): [Network, PlatformToChains<P>] {
  switch (platform) {
    case "Evm":
      // we may get a hex string or a stringified number
      return evmChainIdToNetworkChainPair.get(encoding.bignum.decode(chainId));
    case "Cosmwasm":
      return cosmwasmChainIdToNetworkChainPair.get(chainId);
    case "Solana":
      return solGenesisHashToNetworkChainPair.get(chainId);
    case "Sui":
      return suiChainIdToNetworkChain.get(chainId);
    case "Near":
      return nearChainIdToNetworkChain.get(chainId);
    case "Aptos":
      return aptosChainIdToNetworkChain.get(encoding.bignum.decode(chainId));
    case "Algorand":
      return algorandChainIdToNetworkChain.get(chainId);
  }
  throw new Error("Unrecognized platform: " + platform);
}

export * from "./evm";
export * from "./solana";
export * from "./cosmwasm";
export * from "./sui";
export * from "./aptos";
export * from "./near";
export * from "./algorand";
