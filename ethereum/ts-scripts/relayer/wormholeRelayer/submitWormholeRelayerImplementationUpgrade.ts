import { ChainId } from "@certusone/wormhole-sdk";
import {
  init,
  getOperatingChains,
  getWormholeRelayer,
  ChainInfo,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";

import { extractChainToBeRegisteredFromRegisterChainVaa } from "../helpers/vaa";

import { inspect } from "util";

const processName = "registerWormholeRelayer";
init();
const chains = getOperatingChains();

/**
 * These are wormhole-relayer implementation upgrade VAAs for mainnet.
 */
const implementationUpgradeVaas: Partial<Record<ChainId, string>> = {
  // [chainId:number]: vaa
}

async function run() {
  console.log(`Start! ${processName}`);

  const tasks = await Promise.allSettled(
    chains.map((chain) => {
      const vaa = implementationUpgradeVaas[chain.chainId] as string;
      if (!vaa) {
        throw new Error("No implementation upgrade VAA found for chain " + chain.chainId);
      }
      console.log(`Submitting upgrade VAA ${vaa} to chain ${chain.chainId}`);
      return submitWormholeRelayerUpgradeVaa(chain, vaa)
    }),
  );

  for (const task of tasks) {
    if (task.status === "rejected") {
      console.error(`Register failed: ${inspect(task.reason)}`);
    } else {
      console.log(`Register succeeded for chain ${task.value}`);
    }
  }
}

async function submitWormholeRelayerUpgradeVaa(
  chain: ChainInfo,
  vaa: string,
) {
  console.log(`Upgrading WormholeRelayer in chain ${chain.chainId}...`);

  const wormholeRelayer = await getWormholeRelayer(chain);

  const overrides = await buildOverrides(
    () => wormholeRelayer.estimateGas.submitContractUpgrade(vaa),
    chain,
  );
  const tx = await wormholeRelayer.submitContractUpgrade(vaa, overrides);

  const receipt = await tx.wait();

  if (receipt.status !== 1) {
    throw new Error(
      `Failed to upgrade on chain ${chain.chainId}, tx id: ${tx.hash}`,
    );
  }
  console.log(
    "Successfully upgraded the core relayer contract on " + chain.chainId,
  );
  return chain.chainId;
}

run().then(() => console.log("Done! " + processName));
