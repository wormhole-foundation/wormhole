import { ChainId, EVMChainId } from "@certusone/wormhole-sdk";
import { PricingContext } from "../app";
import {
  CoreRelayer__factory,
  RelayProvider,
  RelayProvider__factory,
} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";
import { getAllChains, getEthersProvider } from "../env";
import { BigNumber } from "ethers";
import { createContext } from "vm";

export type RelayProviderContractState = {
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
  ctx: PricingContext
): Promise<RelayProviderContractState[]> {
  const allChains = getAllChains(ctx);
  const states: RelayProviderContractState[] = [];
  for (const chain of allChains) {
    const state = await pullCurrentPricingState(ctx, chain);
    states.push(state);
  }
  return states;
}

//Retrieves the current prices from a given chain
export async function pullCurrentPricingState(
  ctx: PricingContext,
  chain: ChainId
): Promise<RelayProviderContractState> {
  const relayProviderAddress = ctx.relayProviders[chain as EVMChainId]; //Cast to EVM chain ID type should be safe
  const RelayProvider = RelayProvider__factory.connect(
    relayProviderAddress,
    getEthersProvider(ctx, chain)
  );

  const state = readState(ctx, chain, RelayProvider, getAllChains(ctx));
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
  ctx: PricingContext,
  currentChain: ChainId,
  relayProvider: RelayProvider,
  allChains: ChainId[]
): Promise<RelayProviderContractState | null> {
  ctx.logger.info(
    "Gathering relay provider contract status for chain " + currentChain
  );

  try {
    const rewardAddress = await relayProvider.getRewardAddress();
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
    const owner: string = await relayProvider.owner();

    for (const chain of allChains) {
      supportedChains.push({
        chainId: chain,
        isSupported: await relayProvider.isChainSupported(chain),
      });

      targetChainAddresses.push({
        chainId: chain,
        whAddress: await relayProvider.getTargetChainAddress(chain),
      });

      deliveryOverheads.push({
        chainId: chain,
        deliveryOverhead: await relayProvider.quoteDeliveryOverhead(chain),
      });
      maximumBudgets.push({
        chainId: chain,
        maximumBudget: await relayProvider.quoteMaximumBudget(chain),
      });
      gasPrices.push({
        chainId: chain,
        gasPrice: await relayProvider.quoteGasPrice(chain),
      });
      usdPrices.push({
        chainId: chain,
        usdPrice: await relayProvider.quoteAssetPrice(chain),
      });
      const buffer = await relayProvider.getAssetConversionBuffer(chain);
      assetConversionBuffers.push({
        chainId: chain,
        tolerance: buffer.tolerance,
        toleranceDenominator: buffer.toleranceDenominator,
      });
    }

    return {
      chainId: currentChain,
      contractAddress: relayProvider.address,
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

export function printableState(state: RelayProviderContractState): string {
  let output = "";
  output += "RelayProvider: \n";
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
