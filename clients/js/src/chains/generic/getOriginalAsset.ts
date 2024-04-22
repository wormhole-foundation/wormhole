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
import {
  ChainId,
  ChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACTS } from "../../consts";
import { Network } from "../../utils";
import { impossible } from "../../vaa";
import { getOriginalAssetSei } from "../sei/sdk";
import { getProviderForChain } from "./provider";

export const getOriginalAsset = async (
  chain: ChainId | ChainName,
  network: Network,
  assetAddress: string,
  rpc?: string
): Promise<WormholeWrappedInfo> => {
  const chainName = coalesceChainName(chain);
  const tokenBridgeAddress = CONTRACTS[network][chainName].token_bridge;
  if (!tokenBridgeAddress) {
    throw new Error(
      `Token bridge address not defined for ${chainName} ${network}`
    );
  }

  switch (chainName) {
    case "unset":
      throw new Error("Chain not set");
    case "solana": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSolana(provider, tokenBridgeAddress, assetAddress);
    }
    case "acala":
    case "arbitrum":
    case "aurora":
    case "avalanche":
    case "base":
    case "bsc":
    case "celo":
    case "ethereum":
    case "fantom":
    case "gnosis":
    case "karura":
    case "klaytn":
    case "moonbeam":
    case "neon":
    case "oasis":
    case "optimism":
    case "polygon":
    // case "rootstock":
    case "scroll":
    case "mantle":
    case "blast":
    case "xlayer":
    case "linea":
    case "berachain":
    case "seievm":
    case "sepolia":
    case "arbitrum_sepolia":
    case "base_sepolia":
    case "optimism_sepolia":
    case "polygon_sepolia":
    case "holesky": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetEth(
        tokenBridgeAddress,
        provider,
        assetAddress,
        chain
      );
    }
    case "terra":
    case "terra2": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetTerra(provider, assetAddress);
    }
    case "injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetInjective(assetAddress, provider);
    }
    case "sei": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSei(assetAddress, provider);
    }
    case "xpla": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetXpla(provider, assetAddress);
    }
    case "algorand": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetAlgorand(
        provider,
        BigInt(tokenBridgeAddress),
        BigInt(assetAddress)
      );
    }
    case "near": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetNear(provider, tokenBridgeAddress, assetAddress);
    }
    case "aptos": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetAptos(provider, tokenBridgeAddress, assetAddress);
    }
    case "sui": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSui(provider, tokenBridgeAddress, assetAddress);
    }
    case "btc":
    case "osmosis":
    case "pythnet":
    case "wormchain":
    case "cosmoshub":
    case "evmos":
    case "kujira":
    case "neutron":
    case "celestia":
    case "stargaze":
    case "seda":
    case "dymension":
    case "provenance":
    case "rootstock":
      throw new Error(`${chainName} not supported`);
    default:
      impossible(chainName);
  }
};
