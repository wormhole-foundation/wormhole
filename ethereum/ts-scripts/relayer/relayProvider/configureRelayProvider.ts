import {
  init,
  loadChains,
  ChainInfo,
  loadScriptConfig,
  getRelayProvider,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";

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
  if (!thisChainsConfigInfo.approvedSenders) {
    throw new Error(
      "Failed to find approvedSenders info for chain " + chain.chainId
    );
  }

  console.log("Set address info...");
  await relayProvider.updateRewardAddress(thisChainsConfigInfo.rewardAddress);

  //TODO refactor to use the batch price update, probably
  console.log("Set gas and native prices...");
  for (let i = 0; i < chains.length; i++) {
    const targetChainPriceUpdate = config.pricingInfo.find(
      (x: any) => x.chainId == chains[i].chainId
    );
    if (!targetChainPriceUpdate) {
      throw new Error(
        "Failed to find pricingInfo for chain " + chains[i].chainId
      );
    }
    //delivery addresses are not done by this script, but rather the register chains script.
    await relayProvider
      .updateDeliverGasOverhead(
        chains[i].chainId,
        targetChainPriceUpdate.deliverGasOverhead
      )
      .then(wait);
    await relayProvider
      .updatePrice(
        chains[i].chainId,
        targetChainPriceUpdate.updatePriceGas,
        targetChainPriceUpdate.updatePriceNative
      )
      .then(wait);
    await relayProvider
      .updateMaximumBudget(
        chains[i].chainId,
        targetChainPriceUpdate.maximumBudget
      )
      .then(wait);
    await relayProvider
      .updateAssetConversionBuffer(chains[i].chainId, 5, 100)
      .then(wait);
  }

  console.log("done with registrations on " + chain.chainId);
}

run().then(() => console.log("Done! " + processName));
