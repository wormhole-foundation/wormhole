import {
  WormholeWrappedInfo,
  getOriginalAssetAlgorand,
  getOriginalAssetAptos,
  getOriginalAssetEth,
  getOriginalAssetNear,
  getOriginalAssetSolana,
  getOriginalAssetTerra,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getOriginalAsset";
import { getOriginalAssetSui } from "../../sdk/sui";
import { getOriginalAssetInjective } from "@certusone/wormhole-sdk/lib/esm/token_bridge/injective";
import { ethers } from "ethers";
import { getOriginalAssetSei } from "../sei/sdk";
import { getProviderForChain } from "./provider";
import { Network } from "@wormhole-foundation/sdk-base";
import { toLegacyChainId } from "../../sdk/array";
import {
  CliChainLike,
  cliChainToPlatform,
  getTokenBridgeContract,
  toCliChain,
} from "../../utils";

export const getOriginalAsset = async (
  chain: CliChainLike,
  network: Network,
  assetAddress: string,
  rpc?: string
): Promise<WormholeWrappedInfo> => {
  const chainName = toCliChain(chain);
  const tokenBridgeAddress = getTokenBridgeContract(network, chainName);
  if (!tokenBridgeAddress) {
    throw new Error(
      `Token bridge address not defined for ${chainName} ${network}`
    );
  }

  if (cliChainToPlatform(chainName) === "Evm") {
    const provider = getProviderForChain(chainName, network, {
      rpc,
    }) as ethers.providers.JsonRpcProvider;
    return getOriginalAssetEth(
      tokenBridgeAddress,
      provider,
      assetAddress,
      toLegacyChainId(chain)
    );
  }

  switch (chainName) {
    case "Solana": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getOriginalAssetSolana(provider, tokenBridgeAddress, assetAddress);
    }
    case "Terra2": {
      const provider = getProviderForChain(chainName, network, { rpc });
      // the legacy SDK bundles its own (older) @terra-money/terra.js; the
      // LCD client is runtime-compatible
      return getOriginalAssetTerra(provider as any, assetAddress);
    }
    case "Injective": {
      const provider = getProviderForChain(chainName, network, { rpc });
      // the legacy SDK bundles its own (older) @injectivelabs/sdk-ts; the
      // wasm api client is runtime-compatible
      return getOriginalAssetInjective(assetAddress, provider as any);
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
      return (await getOriginalAssetSui(
        provider,
        tokenBridgeAddress,
        assetAddress
      )) as WormholeWrappedInfo;
    }
    default:
      throw new Error(`${chainName} not supported`);
  }
};
