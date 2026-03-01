import {
  WormholeWrappedInfo,
  getOriginalAssetAlgorand,
  getOriginalAssetAptos,
  getOriginalAssetEth,
  getOriginalAssetNear,
  getOriginalAssetSolana,
  getOriginalAssetSui,
  getOriginalAssetTerra,
  getOriginalAssetXpla,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getOriginalAsset";
import { getOriginalAssetInjective } from "@certusone/wormhole-sdk/lib/esm/token_bridge/injective";
import { impossible } from "../../vaa";
import { getOriginalAssetSei } from "../sei/sdk";
import { getProviderForChain } from "./provider";
import {
  Chain,
  ChainId,
  Network,
  contracts,
  toChain,
  toChainId
} from "@wormhole-foundation/sdk-base";

export const getOriginalAsset = async (
  chain: ChainId | Chain,
  network: Network,
  assetAddress: string,
  rpc?: string
): Promise<WormholeWrappedInfo> => {
  const chainName = toChain(chain);
  const tokenBridgeAddress = contracts.tokenBridge.get(network, chainName);
  if (!tokenBridgeAddress) {
    throw new Error(
      `Token bridge address not defined for ${chainName} ${network}`
    );
  }

  switch (chainName) {
    case "Solana": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSolana(provider, tokenBridgeAddress, assetAddress);
    }
    case "Arbitrum":
    case "Avalanche":
    case "Base":
    case "Bsc":
    case "Celo":
    case "Ethereum":
    case "Fantom":
    case "Klaytn":
    case "Moonbeam":
    case "Optimism":
    case "Polygon":
    case "Scroll":
    case "Mantle":
    case "Xlayer":
    case "Linea":
    case "Berachain":
    case "Seievm":
    case "Sepolia":
    case "ArbitrumSepolia":
    case "BaseSepolia":
    case "OptimismSepolia":
    case "PolygonSepolia":
    case "Holesky": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetEth(
        tokenBridgeAddress,
        provider,
        assetAddress,
        // @ts-ignore: legacy chain ids
        toChainId(chain)
      );
    }
    case "Injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetInjective(assetAddress, provider);
    }
    case "Sei": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSei(assetAddress, provider);
    }
    case "Algorand": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetAlgorand(
        provider,
        BigInt(tokenBridgeAddress),
        BigInt(assetAddress)
      );
    }
    case "Near": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetNear(provider, tokenBridgeAddress, assetAddress);
    }
    case "Aptos": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetAptos(provider, tokenBridgeAddress, assetAddress);
    }
    case "Sui": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSui(provider, tokenBridgeAddress, assetAddress);
    }
    case "Btc":
    case "Osmosis":
    case "Pythnet":
    case "Wormchain":
    case "Cosmoshub":
    case "Evmos":
    case "Kujira":
    case "Neutron":
    case "Celestia":
    case "Stargaze":
    case "Seda":
    case "Dymension":
    case "Provenance":
      throw new Error(`${chainName} not supported`);
    default:
      // @ts-ignore: unsupported
      impossible(chainName);
  }
};
