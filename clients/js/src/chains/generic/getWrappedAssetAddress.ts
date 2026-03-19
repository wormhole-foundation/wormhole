import {
  getForeignAssetAlgorand,
  getForeignAssetAptos,
  getForeignAssetEth,
  getForeignAssetNear,
  getForeignAssetSolana,
  getForeignAssetSui,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getForeignAsset";
import { getForeignAssetInjective } from "@certusone/wormhole-sdk/lib/esm/token_bridge/injective";
import { impossible } from "../../vaa";
import { getForeignAssetSei } from "../sei/sdk";
import { getProviderForChain } from "./provider";
import {
  Chain,
  ChainId,
  Network,
  contracts,
  toChain,
  toChainId,
} from "@wormhole-foundation/sdk-base";
import { tryNativeToUint8Array } from "../../sdk/array";
import { castChainIdToOldSdk } from "../../utils";

export const getWrappedAssetAddress = async (
  chain: ChainId | Chain,
  network: Network,
  originChain: ChainId | Chain,
  originAddress: string,
  rpc?: string
): Promise<string | null> => {
  const chainName = toChain(chain);
  const originAddressUint8Array = tryNativeToUint8Array(
    originAddress,
    originChain
  );
  const tokenBridgeAddress = contracts.tokenBridge.get(network, chainName);
  if (!tokenBridgeAddress) {
    throw new Error(
      `Token bridge address not defined for ${chainName} ${network}`
    );
  }

  switch (chainName) {
    case "Solana": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSolana(
        provider,
        tokenBridgeAddress,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddressUint8Array
      );
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
      return getForeignAssetEth(
        tokenBridgeAddress,
        provider,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddressUint8Array
      );
    }
    case "Injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetInjective(
        tokenBridgeAddress,
        provider,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddressUint8Array
      );
    }
    case "Sei": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSei(
        tokenBridgeAddress,
        provider,
        toChainId(originChain),
        originAddressUint8Array
      );
    }
    case "Algorand": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetAlgorand(
        provider,
        BigInt(tokenBridgeAddress),
        castChainIdToOldSdk(toChainId(originChain)),
        originAddress
      ).then((x) => x?.toString() ?? null);
    }
    case "Near": {
      const provider = await getProviderForChain(chainName, network, { rpc });
      return getForeignAssetNear(
        provider,
        tokenBridgeAddress,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddress
      );
    }
    case "Aptos": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetAptos(
        provider,
        tokenBridgeAddress,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddress
      );
    }
    case "Sui": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSui(
        provider,
        tokenBridgeAddress,
        castChainIdToOldSdk(toChainId(originChain)),
        originAddressUint8Array
      );
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
    case "Fogo":
    case "Unichain":
    case "HyperCore":
    case "Worldchain":
    case "Ink":
    case "HyperEVM":
    case "Monad":
    case "Mezo":
    case "Sonic":
    case "Converge":
    case "Plume":
    case "XRPLEVM":
    case "Plasma":
    case "CreditCoin":
    case "Noble":
      throw new Error(`${chainName} not supported`);
    default:
      impossible(chainName);
  }
};
