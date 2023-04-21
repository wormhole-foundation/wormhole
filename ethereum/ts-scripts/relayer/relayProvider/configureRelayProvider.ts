import { ChainId, tryNativeToHexString } from "@certusone/wormhole-sdk";
import type { BigNumberish } from "ethers";
import {
  init,
  loadChains,
  ChainInfo,
  loadScriptConfig,
  getCoreRelayerAddress,
  getRelayProvider,
  getRelayProviderAddress,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";

import type { RelayProviderStructs } from "../../../ethers-contracts/RelayProvider";

/**
 * Meant for `config.pricingInfo`
 */
interface PricingInfo {
  chainId: ChainId
  deliverGasOverhead: BigNumberish
  updatePriceGas: BigNumberish
  updatePriceNative: BigNumberish
  maximumBudget: BigNumberish
};

const processName = "configureRelayProvider";
init();
const operatingChains = getOperatingChains();
const chains = loadChains();
const config = loadScriptConfig(processName);

async function run() {
  console.log("Start! " + processName);

  for (let i = 0; i < operatingChains.length; i++) {
    await configureChainsRelayProvider(chains[i]);
  }
}

async function configureChainsRelayProvider(chain: ChainInfo) {
  console.log("about to perform RelayProvider configuration for chain " + chain.chainId);
  const relayProvider = getRelayProvider(chain);
  const coreRelayer = await getCoreRelayerAddress(chain);

  const thisChainsConfigInfo = config.addresses.find(
    (x: any) => x.chainId == chain.chainId
  );

  if (!thisChainsConfigInfo) {
    throw new Error(
      "Failed to find address config info for chain " + chain.chainId
    );
  }
  if (!thisChainsConfigInfo.rewardAddress) {
    throw new Error(
      "Failed to find reward address info for chain " + chain.chainId
    );
  }

  const coreConfig: RelayProviderStructs.CoreConfigStruct = {
    updateCoreRelayer: true,
    updateRewardAddress: true,
    coreRelayer,
    rewardAddress: thisChainsConfigInfo.rewardAddress,
  };
  const updates: RelayProviderStructs.UpdateStruct[] = [];

  // Set the entire relay provider configuration
  for (const targetChain of chains) {
    const targetChainPriceUpdate = (config.pricingInfo as PricingInfo[]).find(
      (x: any) => x.chainId == targetChain.chainId
    );
    if (!targetChainPriceUpdate) {
      throw new Error(
        "Failed to find pricingInfo for chain " + targetChain.chainId
      );
    }
    const targetChainProviderAddress = getRelayProviderAddress(targetChain);
    const remoteRelayProvider =
      "0x" + tryNativeToHexString(targetChainProviderAddress, "ethereum");
    const chainConfigUpdate = {
      chainId: targetChain.chainId,
      updateAssetConversionBuffer: true,
      updateDeliverGasOverhead: true,
      updatePrice: true,
      updateMaximumBudget: true,
      updateTargetChainAddress: true,
      updateSupportedChain: true,
      isSupported: true,
      buffer: 5,
      bufferDenominator: 100,
      newWormholeFee: 0,
      newGasOverhead: targetChainPriceUpdate.deliverGasOverhead,
      gasPrice: targetChainPriceUpdate.updatePriceGas,
      nativeCurrencyPrice: targetChainPriceUpdate.updatePriceNative,
      targetChainAddress: remoteRelayProvider,
      maximumTotalBudget: targetChainPriceUpdate.maximumBudget,
    };
    updates.push(chainConfigUpdate);
  }
  await relayProvider.updateConfig(updates, coreConfig).then(wait);

  console.log("done with RelayProvider configuration on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
