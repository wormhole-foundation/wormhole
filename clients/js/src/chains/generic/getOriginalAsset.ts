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
} from "@wormhole-foundation/sdk-base";
import { toChainId } from "@wormhole-foundation/sdk";

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
    case "Acala":
    case "Arbitrum":
    case "Aurora":
    case "Avalanche":
    case "Base":
    case "Bsc":
    case "Celo":
    case "Ethereum":
    case "Fantom":
    case "Gnosis":
    case "Karura":
    case "Klaytn":
    case "Moonbeam":
    case "Neon":
    case "Oasis":
    case "Optimism":
    case "Polygon":
    case "Scroll":
    case "Mantle":
    case "Blast":
    case "Xlayer":
    case "Linea":
    case "Berachain":
    case "Snaxchain":
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
        toChainId(chain)
      );
    }
    case "Terra":
    case "Terra2": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetTerra(provider, assetAddress);
    }
    case "Injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetInjective(assetAddress, provider);
    }
    case "Sei": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSei(assetAddress, provider);
    }
    case "Xpla": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetXpla(provider, assetAddress);
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
    case "Rootstock":
      throw new Error(`${chainName} not supported`);
    default:
      impossible(chainName);
  }
};
