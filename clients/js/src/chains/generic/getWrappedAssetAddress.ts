import {
  getForeignAssetAlgorand,
  getForeignAssetAptos,
  getForeignAssetEth,
  getForeignAssetNear,
  getForeignAssetSolana,
  getForeignAssetSui,
  getForeignAssetTerra,
  getForeignAssetXpla,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getForeignAsset";
import { getForeignAssetInjective } from "@certusone/wormhole-sdk/lib/esm/token_bridge/injective";
import { tryNativeToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils/array";
import {
  ChainId,
  ChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACTS } from "../../consts";
import { Network } from "../../utils";
import { impossible } from "../../vaa";
import { getForeignAssetSei } from "../sei/sdk";
import { getProviderForChain } from "./provider";

export const getWrappedAssetAddress = async (
  chain: ChainId | ChainName,
  network: Network,
  originChain: ChainId | ChainName,
  originAddress: string,
  rpc?: string
): Promise<string | null> => {
  const chainName = coalesceChainName(chain);
  const originAddressUint8Array = tryNativeToUint8Array(
    originAddress,
    originChain
  );
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
      return getForeignAssetSolana(
        provider,
        tokenBridgeAddress,
        originChain,
        originAddressUint8Array
      );
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
      return getForeignAssetEth(
        tokenBridgeAddress,
        provider,
        originChain,
        originAddressUint8Array
      );
    }
    case "terra":
    case "terra2": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetTerra(
        tokenBridgeAddress,
        provider,
        originChain,
        originAddressUint8Array
      );
    }
    case "injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetInjective(
        tokenBridgeAddress,
        provider,
        originChain,
        originAddressUint8Array
      );
    }
    case "sei": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSei(
        tokenBridgeAddress,
        provider,
        originChain,
        originAddressUint8Array
      );
    }
    case "xpla": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetXpla(
        tokenBridgeAddress,
        provider,
        originChain,
        originAddressUint8Array
      );
    }
    case "algorand": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetAlgorand(
        provider,
        BigInt(tokenBridgeAddress),
        originChain,
        originAddress
      ).then((x) => x?.toString() ?? null);
    }
    case "near": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getForeignAssetNear(
        provider,
        tokenBridgeAddress,
        originChain,
        originAddress
      );
    }
    case "aptos": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetAptos(
        provider,
        tokenBridgeAddress,
        originChain,
        originAddress
      );
    }
    case "sui": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSui(
        provider,
        tokenBridgeAddress,
        originChain,
        originAddressUint8Array
      );
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
    case "rootstock":
    case "stargaze":
    case "seda":
    case "dymension":
    case "provenance":
      throw new Error(`${chainName} not supported`);
    default:
      impossible(chainName);
  }
};
