import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { BigNumber } from "ethers";

import {
  init,
  loadChains,
  ChainInfo,
  getCoreRelayerAddress,
  getRelayProvider,
  getRelayProviderAddress,
  getProvider,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";
import { wait } from "../helpers/utils";

const processName = "readRelayProviderContractState";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start! " + processName);

  const states: any = [];

  for (let i = 0; i < chains.length; i++) {
    const state = await readState(chains[i]);
    if (state) {
      printState(state);
      states.push(state);
    }
  }

  writeOutputFiles(states, processName);
}

type RelayProviderContractState = {
  chainId: number;
  contractAddress: string;
  rewardAddress: string;
  providerAddresses: { chainId: number; providerAddress: string }[];
  deliveryOverheads: { chainId: number; deliveryOverhead: BigNumber }[];
  maximumBudgets: { chainId: number; maximumBudget: BigNumber }[];
  gasPrices: { chainId: number; gasPrice: BigNumber }[];
  usdPrices: { chainId: number; usdPrice: BigNumber }[];
  assetConversionBuffers: {
    chainId: number;
    tolerance: number;
    toleranceDenominator: number;
  }[];
  owner: string;
};

async function readState(
  chain: ChainInfo
): Promise<RelayProviderContractState | null> {
  console.log(
    "Gathering relay provider contract status for chain " + chain.chainId
  );

  try {
    const relayProvider = getRelayProvider(chain, getProvider(chain));
    const contractAddress = getRelayProviderAddress(chain);
    const rewardAddress = await relayProvider.getRewardAddress();
    const providerAddresses: {
      chainId: number;
      providerAddress: string;
    }[] = [];
    const deliveryOverheads: {
      chainId: number;
      deliveryOverhead: BigNumber;
    }[] = [];
    const maximumBudgets: { chainId: number; maximumBudget: BigNumber }[] = [];
    const gasPrices: { chainId: number; gasPrice: BigNumber }[] = [];
    const usdPrices: { chainId: number; usdPrice: BigNumber }[] = [];
    const assetConversionBuffers: {
      chainId: number;
      tolerance: number;
      toleranceDenominator: number;
    }[] = [];
    const owner: string = await relayProvider.owner();

    for (const chainInfo of chains) {
      //TODO
      // providerAddresses.push({
      //   chainId: chainInfo.chainId,
      //   providerAddress: (
      //     await relayProvider.getDeliveryAddress(chainInfo.chainId)
      //   ).toString(),
      // });
      deliveryOverheads.push({
        chainId: chainInfo.chainId,
        deliveryOverhead: await relayProvider.quoteDeliveryOverhead(
          chainInfo.chainId
        ),
      });
      maximumBudgets.push({
        chainId: chainInfo.chainId,
        maximumBudget: await relayProvider.quoteMaximumBudget(
          chainInfo.chainId
        ),
      });
      gasPrices.push({
        chainId: chainInfo.chainId,
        gasPrice: await relayProvider.quoteGasPrice(chainInfo.chainId),
      });
      usdPrices.push({
        chainId: chainInfo.chainId,
        usdPrice: await relayProvider.quoteAssetPrice(chainInfo.chainId),
      });
      const buffer = await relayProvider.getAssetConversionBuffer(
        chainInfo.chainId
      );
      assetConversionBuffers.push({
        chainId: chainInfo.chainId,
        tolerance: buffer.tolerance,
        toleranceDenominator: buffer.toleranceDenominator,
      });
    }

    return {
      chainId: chain.chainId,
      contractAddress,
      rewardAddress,
      providerAddresses,
      deliveryOverheads,
      maximumBudgets,
      gasPrices,
      usdPrices,
      assetConversionBuffers,
      owner,
    };
  } catch (e) {
    console.error(e);
    console.log("Failed to gather status for chain " + chain.chainId);
  }

  return null;
}

function printState(state: RelayProviderContractState) {
  console.log("");
  console.log("RelayProvider: ");
  printFixed("Chain ID: ", state.chainId.toString());
  printFixed("Contract Address:", state.contractAddress);
  printFixed("Owner Address:", state.owner);
  printFixed("Reward Address:", state.rewardAddress);

  console.log("");

  printFixed("Registered Providers", "");
  state.providerAddresses.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.providerAddress);
  });
  console.log("");

  printFixed("Delivery Overheads", "");
  state.deliveryOverheads.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.deliveryOverhead.toString());
  });
  console.log("");

  printFixed("Gas Prices", "");
  state.gasPrices.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.gasPrice.toString());
  });
  console.log("");

  printFixed("USD Prices", "");
  state.usdPrices.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.usdPrice.toString());
  });
  console.log("");

  printFixed("Maximum Budgets", "");
  state.maximumBudgets.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.maximumBudget.toString());
  });
  console.log("");

  printFixed("Asset Conversion Buffers", "");
  state.assetConversionBuffers.forEach((x) => {
    printFixed("  Chain: " + x.chainId, "");
    printFixed("    Tolerance: ", x.tolerance.toString());
    printFixed("    Denominator: ", x.toleranceDenominator.toString());
  });
  console.log("");
}

function printFixed(title: string, content: string) {
  const length = 80;
  const spaces = length - title.length - content.length;
  let str = "";
  if (spaces > 0) {
    for (let i = 0; i < spaces; i++) {
      str = str + " ";
    }
  }
  console.log(title + str + content);
}

run().then(() => console.log("Done! " + processName));
