import {
  CHAINS,
  ChainName,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { relayer, Network } from "@certusone/wormhole-sdk";
import yargs, { string } from "yargs";
import { CONTRACTS, NETWORKS } from "../consts";
import { assertNetwork } from "../utils";
import { impossible } from "../vaa";
import { ethers } from "ethers";

export const command = "status <network> <chain> <tx>";
export const desc =
  "Prints information about the automatic delivery initiated on the specified network, chain, and tx";
export const builder = (y: typeof yargs) =>
  y
    .positional("network", {
      describe: "Network",
      choices: ["mainnet", "testnet", "devnet"],
      demandOption: true,
    } as const)
    .positional("chain", {
      describe: "Source chain",
      choices: Object.keys(CHAINS) as ChainName[],
      demandOption: true,
    } as const)
    .positional("tx", {
      describe: "Source transaction hash",
      type: "string",
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = argv.network.toUpperCase();
  assertNetwork(network);
  const chain = argv.chain;
  assertChain(chain);

  const addr =
    relayer.RELAYER_CONTRACTS[network][chain]?.wormholeRelayerAddress;
  if (!addr) {
    throw new Error(`Wormhole Relayer not deployed on ${chain} in ${network}`);
  }

  const sourceRPC = NETWORKS[network as Network][chain as ChainName].rpc;
  const sourceChainProvider = new ethers.providers.JsonRpcProvider(sourceRPC);
  const targetChainProviders = new Map<ChainName, ethers.providers.Provider>();
  for (const key in NETWORKS[network]) {
    targetChainProviders.set(
      key as ChainName,
      new ethers.providers.JsonRpcProvider(
        NETWORKS[network as Network][key as ChainName].rpc
      )
    );
  }

  const info = await relayer.getWormholeRelayerInfo(chain, argv.tx, {
    environment: network,
    sourceChainProvider,
    targetChainProviders,
  });

  console.log(relayer.stringifyWormholeRelayerInfo(info));
};
