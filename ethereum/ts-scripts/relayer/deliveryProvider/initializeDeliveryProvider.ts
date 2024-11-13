import type { BigNumberish } from "ethers";
import {
  init,
  loadChains,
  ChainInfo,
  loadScriptConfig,
  getDeliveryProvider,
  getDeliveryProviderAddress,
  getOperatingChains,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { nativeEthereumAddressToHex, wait } from "../helpers/utils";

import type { DeliveryProviderStructs } from "../../../ethers-contracts/DeliveryProvider";
import { ChainId } from "@wormhole-foundation/sdk-base"

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

const processName = "initializeDeliveryProvider";
init();
const operatingChains = getOperatingChains();
const allChains = loadChains();
const config = loadScriptConfig(processName);

async function run() {
  console.log("Start! " + processName);

  for (const chain of operatingChains) {
    await configureChainsDeliveryProvider(chain);
  }
}

async function configureChainsDeliveryProvider(chain: ChainInfo) {
  console.log(
    "about to perform DeliveryProvider configuration for chain " + chain.chainId
  );
  const deliveryProvider = await getDeliveryProvider(chain);

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

  const VAA_KEY_TYPE = 1 << 0;
  // const CCTP_KEY_TYPE = 1 << 1;
  const coreConfig: DeliveryProviderStructs.CoreConfigStruct = {
    updateWormholeRelayer: false,
    updateRewardAddress: true,
    updateSupportedKeyTypes: true,
    coreRelayer: "0x0000000000000000000000000000000000000000",
    rewardAddress: thisChainsConfigInfo.rewardAddress,
    supportedKeyTypesBitmap: VAA_KEY_TYPE,
  };
  const updates: DeliveryProviderStructs.UpdateStruct[] = [];

  // Set the entire relay provider configuration
  for (const targetChain of allChains) {
    const targetChainPriceUpdate = (config.pricingInfo as PricingInfo[]).find(
      (x) => x.chainId == targetChain.chainId
    );
    if (!targetChainPriceUpdate) {
      throw new Error(
        "Failed to find pricingInfo for chain " + targetChain.chainId
      );
    }
    const targetChainProviderAddress = getDeliveryProviderAddress(targetChain);
    const remoteDeliveryProvider = nativeEthereumAddressToHex(targetChainProviderAddress);
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
