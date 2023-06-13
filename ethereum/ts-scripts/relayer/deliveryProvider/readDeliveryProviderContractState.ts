import { BigNumber, ethers } from "ethers";

import {
  init,
  ChainInfo,
  getDeliveryProvider,
  getDeliveryProviderAddress,
  getProvider,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";

const processName = "readDeliveryProviderContractState";
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

type DeliveryProviderContractState = {
  chainId: number;
  contractAddress: string;
  rewardAddress: string;
  deliveryOverheads: { chainId: number; deliveryOverhead: BigNumber }[];
  supportedChains: { chainId: number; isSupported: boolean }[];
  targetChainAddresses: { chainId: number; whAddress: string }[];
  gasPrices: { chainId: number; gasPrice: BigNumber }[];
  weiPrices: { chainId: number; weiPrice: BigNumber }[];
  owner: string;
};

async function readState(
  chain: ChainInfo
): Promise<DeliveryProviderContractState | null> {
  console.log(
    "Gathering relay provider contract status for chain " + chain.chainId
  );

  try {
    const deliveryProvider = getDeliveryProvider(chain, getProvider(chain));
    const contractAddress = getDeliveryProviderAddress(chain);
    console.log("Querying Relay Provider for code");
    const provider = getProvider(chain);
    const codeReceipt = await provider.getCode(contractAddress);
    console.log("Code: " + codeReceipt);
    const rewardAddress = await deliveryProvider.getRewardAddress();
    const supportedChains: {
      chainId: number;
      isSupported: boolean;
    }[] = [];
    const targetChainAddresses: {
      chainId: number;
      whAddress: string;
    }[] = [];
    const deliveryOverheads: {
      chainId: number;
      deliveryOverhead: BigNumber;
    }[] = [];
    const gasPrices: { chainId: number; gasPrice: BigNumber }[] = [];
    const weiPrices: { chainId: number; weiPrice: BigNumber }[] = [];
    const owner: string = await deliveryProvider.owner();

    for (const chainInfo of chains) {
      supportedChains.push({
        chainId: chainInfo.chainId,
        isSupported: await deliveryProvider.isChainSupported(chainInfo.chainId),
      });

      targetChainAddresses.push({
        chainId: chainInfo.chainId,
        whAddress: await deliveryProvider.getTargetChainAddress(
          chainInfo.chainId
        ),
      });

      deliveryOverheads.push({
        chainId: chainInfo.chainId,
        deliveryOverhead: await deliveryProvider.quoteDeliveryOverhead(
          chainInfo.chainId
        ),
      });
      gasPrices.push({
        chainId: chainInfo.chainId,
        gasPrice: await deliveryProvider.quoteGasPrice(chainInfo.chainId),
      });
      weiPrices.push({
        chainId: chainInfo.chainId,
        weiPrice: await deliveryProvider.quoteAssetConversion(
          chainInfo.chainId,
          ethers.utils.parseEther("1")
        ),
      });
    }

    return {
      chainId: chain.chainId,
      contractAddress,
      rewardAddress,
      deliveryOverheads,
      supportedChains,
      targetChainAddresses,
      gasPrices,
      weiPrices,
      owner,
    };
  } catch (e) {
    console.error(e);
    console.log("Failed to gather status for chain " + chain.chainId);
  }

  return null;
}

function printState(state: DeliveryProviderContractState) {
  console.log("");
  console.log("DeliveryProvider: ");
  printFixed("Chain ID: ", state.chainId.toString());
  printFixed("Contract Address:", state.contractAddress);
  printFixed("Owner Address:", state.owner);
  printFixed("Reward Address:", state.rewardAddress);

  console.log("");

  printFixed("Supported Chains", "");
  state.supportedChains.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.isSupported.toString());
  });
  console.log("");

  printFixed("Target Chain Addresses", "");
  state.targetChainAddresses.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.whAddress.toString());
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
  state.weiPrices.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.weiPrice.toString());
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
