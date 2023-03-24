import type { ChainId } from "@certusone/wormhole-sdk";
import type { BigNumberish } from "ethers";
import {
  init,
  loadChains,
  ChainInfo,
  loadScriptConfig,
  getRelayProvider,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";

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

/**
 * Must match `RelayProviderStructs.UpdatePrice`
 */
interface UpdatePrice {
  chainId: ChainId
  gasPrice: BigNumberish
  nativeCurrencyPrice: BigNumberish
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
  console.log("about to perform configurations for chain " + chain.chainId);

  const relayProvider = getRelayProvider(chain);
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

  console.log("Set address info...");
  await relayProvider.updateRewardAddress(thisChainsConfigInfo.rewardAddress);

  console.log("Set gas and native prices...");

  // Batch update prices
  const pricingUpdates: UpdatePrice[] = (config.pricingInfo as PricingInfo[]).map((info) => {
    return {
      chainId: info.chainId,
      gasPrice: info.updatePriceGas,
      nativeCurrencyPrice: info.updatePriceNative,
    };
  });
  await relayProvider.updatePrices(pricingUpdates).then(wait);

  // Set the rest of the relay provider configuration
  for (const targetChain of chains) {
    const targetChainPriceUpdate = config.pricingInfo.find(
      (x: any) => x.chainId == targetChain.chainId
    );
    if (!targetChainPriceUpdate) {
      throw new Error(
        "Failed to find pricingInfo for chain " + targetChain.chainId
      );
    }
    //delivery addresses are not done by this script, but rather the register chains script.
    await relayProvider
      .updateDeliverGasOverhead(
        targetChain.chainId,
        targetChainPriceUpdate.deliverGasOverhead
      )
      .then(wait);
    await relayProvider
      .updateMaximumBudget(
        targetChain.chainId,
        targetChainPriceUpdate.maximumBudget
      )
      .then(wait);
    await relayProvider
      .updateAssetConversionBuffer(targetChain.chainId, 5, 100)
      .then(wait);
  }

  console.log("done with registrations on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
