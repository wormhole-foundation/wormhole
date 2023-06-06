import { ChainId, EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "../app";
import {
  WormholeRelayer__factory,
  DeliveryProvider,
  DeliveryProvider__factory,
} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";
import { BigNumber } from "ethers";
import { ethers } from "ethers";
import { createContext } from "vm";

export function getAllChains(ctx: GRContext): EVMChainId[] {
  return Object.keys(ctx.deliveryProviders).map((x) =>
    Number(x)
  ) as EVMChainId[];
}

export function getEthersProvider(
  ctx: GRContext,
  chainId: ChainId
): ethers.providers.JsonRpcProvider {
  const rpc = ctx.providers.evm[chainId as EVMChainId];
  if (rpc == undefined || rpc.length == 0) {
    throw new Error(`No rpc found for chainId ${chainId}`);
  }

  return rpc[0];
}
export type DeliveryProviderContractState = {
  chainId: number;
  contractAddress: string;
  rewardAddress: string;
  deliveryOverheads: { chainId: number; deliveryOverhead: BigNumber }[];
  supportedChains: { chainId: number; isSupported: boolean }[];
  targetChainAddresses: { chainId: number; whAddress: string }[];
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

export async function pullAllCurrentPricingStates(
  ctx: GRContext
): Promise<DeliveryProviderContractState[]> {
  const allChains = getAllChains(ctx);
  const states: DeliveryProviderContractState[] = [];
  for (const chain of allChains) {
    const state = await pullCurrentPricingState(ctx, chain);
    states.push(state);
  }
  return states;
}

//Retrieves the current prices from a given chain
export async function pullCurrentPricingState(
  ctx: GRContext,
  chain: ChainId
): Promise<DeliveryProviderContractState> {
  const deliveryProviderAddress = ctx.deliveryProviders[chain as EVMChainId]; //Cast to EVM chain ID type should be safe
  const DeliveryProvider = DeliveryProvider__factory.connect(
    deliveryProviderAddress,
    getEthersProvider(ctx, chain)
  );

  const state = readState(ctx, chain, DeliveryProvider, getAllChains(ctx));
  if (state == null) {
    throw new Error("Could not read state for chain " + chain);
  } else {
    //typscript linter is complaining about this line, but it should be safe
    //@ts-ignore
    return state;
  }
}

//This code is very similar to that in ethereum/ts-scripts/relayer/relayProvider/readRelayProviderContractState.ts,
//It should be considered for a furture refactor

async function readState(
  ctx: GRContext,
  currentChain: ChainId,
  deliveryProvider: DeliveryProvider,
  allChains: ChainId[]
): Promise<DeliveryProviderContractState | null> {
  ctx.logger.info(
    "Gathering relay provider contract status for chain " + currentChain
  );

  try {
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
    const maximumBudgets: { chainId: number; maximumBudget: BigNumber }[] = [];
    const gasPrices: { chainId: number; gasPrice: BigNumber }[] = [];
    const usdPrices: { chainId: number; usdPrice: BigNumber }[] = [];
    const assetConversionBuffers: {
      chainId: number;
      tolerance: number;
      toleranceDenominator: number;
    }[] = [];
    const owner: string = await deliveryProvider.owner();

    for (const chain of allChains) {
      supportedChains.push({
        chainId: chain,
        isSupported: await deliveryProvider.isChainSupported(chain),
      });

      targetChainAddresses.push({
        chainId: chain,
        whAddress: await deliveryProvider.getTargetChainAddress(chain),
      });

      deliveryOverheads.push({
        chainId: chain,
        deliveryOverhead: await deliveryProvider.quoteDeliveryOverhead(chain),
      });
      maximumBudgets.push({
        chainId: chain,
        maximumBudget: await deliveryProvider.maximumBudget(chain),
      });
      gasPrices.push({
        chainId: chain,
        gasPrice: await deliveryProvider.quoteGasPrice(chain),
      });
      usdPrices.push({
        chainId: chain,
        usdPrice: await deliveryProvider.nativeCurrencyPrice(chain),
      });
      const buffer = await deliveryProvider.assetConversionBuffer(chain);
      assetConversionBuffers.push({
        chainId: chain,
        tolerance: buffer.tolerance,
        toleranceDenominator: buffer.toleranceDenominator,
      });
    }

    return {
      chainId: currentChain,
      contractAddress: deliveryProvider.address,
      rewardAddress,
      deliveryOverheads,
      supportedChains,
      targetChainAddresses,
      maximumBudgets,
      gasPrices,
      usdPrices,
      assetConversionBuffers,
      owner,
    };
  } catch (e) {
    ctx.logger.error(e);
    ctx.logger.error("Failed to gather status for chain " + currentChain);
  }

  return null;
}

export function printableState(state: DeliveryProviderContractState): string {
  let output = "";
  output += "DeliveryProvider: \n";
  output += printFixed("Chain ID: ", state.chainId.toString());
  output += printFixed("Contract Address:", state.contractAddress);
  output += printFixed("Owner Address:", state.owner);
  output += printFixed("Reward Address:", state.rewardAddress);

  output += "\n";

  output += printFixed("Supported Chains", "");
  state.supportedChains.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, x.isSupported.toString());
  });
  output += "\n";

  output += printFixed("Target Chain Addresses", "");
  state.targetChainAddresses.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, x.whAddress.toString());
  });
  output += "\n";

  output += printFixed("Delivery Overheads", "");
  state.deliveryOverheads.forEach((x) => {
    output += printFixed(
      "  Chain: " + x.chainId,
      x.deliveryOverhead.toString()
    );
  });
  output += "\n";

  output += printFixed("Gas Prices", "");
  state.gasPrices.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, x.gasPrice.toString());
  });
  output += "\n";

  output += printFixed("USD Prices", "");
  state.usdPrices.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, x.usdPrice.toString());
  });
  output += "\n";

  output += printFixed("Maximum Budgets", "");
  state.maximumBudgets.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, x.maximumBudget.toString());
  });
  output += "\n";

  output += printFixed("Asset Conversion Buffers", "");
  state.assetConversionBuffers.forEach((x) => {
    output += printFixed("  Chain: " + x.chainId, "");
    output += printFixed("    Tolerance: ", x.tolerance.toString());
    output += printFixed(
      "    Denominator: ",
      x.toleranceDenominator.toString()
    );
  });
  output += "\n";

  return output;
}

export function printFixed(title: string, content: string): string {
  const length = 80;
  const spaces = length - title.length - content.length;
  let str = "";
  if (spaces > 0) {
    for (let i = 0; i < spaces; i++) {
      str = str + " ";
    }
  }
  return title + str + content + "\n";
}
