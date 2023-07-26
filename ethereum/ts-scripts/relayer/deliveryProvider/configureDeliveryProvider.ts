import { ChainId, tryNativeToHexString } from "@certusone/wormhole-sdk";
import type { BigNumberish } from "ethers";
import {
  init,
  loadChains,
  ChainInfo,
  loadScriptConfig,
  getWormholeRelayerAddress,
  getDeliveryProvider,
  getDeliveryProviderAddress,
  getOperatingChains,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { wait } from "../helpers/utils";

import type { DeliveryProviderStructs } from "../../../ethers-contracts/DeliveryProvider";

/**
 * Meant for `config.pricingInfo`
 */
interface PricingInfo {
  chainId: ChainId;
  deliverGasOverhead: BigNumberish;
  updatePriceGas: BigNumberish;
  updatePriceNative: BigNumberish;
  maximumBudget: BigNumberish;
}

const processName = "configureDeliveryProvider";
init();
const operatingChains = getOperatingChains();
const chains = loadChains();
const config = loadScriptConfig(processName);

async function run() {
  console.log("Start! " + processName);

  for (let i = 0; i < operatingChains.length; i++) {
    await configureChainsDeliveryProvider(chains[i]);
  }
}

async function configureChainsDeliveryProvider(chain: ChainInfo) {
  console.log(
    "about to perform DeliveryProvider configuration for chain " + chain.chainId
  );
  const deliveryProvider = await getDeliveryProvider(chain);
  const coreRelayer = await getWormholeRelayerAddress(chain);

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

  const coreConfig: DeliveryProviderStructs.CoreConfigStruct = {
    updateWormholeRelayer: true,
    updateRewardAddress: true,
    coreRelayer,
    rewardAddress: thisChainsConfigInfo.rewardAddress,
  };
  const updates: DeliveryProviderStructs.UpdateStruct[] = [];

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
    const targetChainProviderAddress = getDeliveryProviderAddress(targetChain);
    const remoteDeliveryProvider =
      "0x" + tryNativeToHexString(targetChainProviderAddress, "ethereum");
    const chainConfigUpdate: DeliveryProviderStructs.UpdateStruct = {
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
      newGasOverhead: targetChainPriceUpdate.deliverGasOverhead,
      gasPrice: targetChainPriceUpdate.updatePriceGas,
      nativeCurrencyPrice: targetChainPriceUpdate.updatePriceNative,
      targetChainAddress: remoteDeliveryProvider,
      maximumTotalBudget: targetChainPriceUpdate.maximumBudget,
    };
    updates.push(chainConfigUpdate);
  }

  const overrides = await buildOverrides(
    () => deliveryProvider.estimateGas.updateConfig(updates, coreConfig),
    chain
  );
  await deliveryProvider
    .updateConfig(updates, coreConfig, overrides)
    .then(wait);

  console.log("done with DeliveryProvider configuration on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
