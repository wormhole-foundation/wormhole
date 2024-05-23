import yargs from "yargs";
import { ethers } from "ethers";
import { NETWORKS } from "../consts";
import { chainToChain, getNetwork } from "../utils";
import { Chain, assertChain, contracts } from "@wormhole-foundation/sdk-base";
import { relayer } from "@certusone/wormhole-sdk";

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
      type: "string",
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
  const network = getNetwork(argv.network);
  const chain = chainToChain(argv.chain);
  assertChain(chain);

  const addr = contracts.relayer.get(network, chain);
  if (!addr) {
    throw new Error(`Wormhole Relayer not deployed on ${chain} in ${network}`);
  }

  const sourceRPC = NETWORKS[network][chain].rpc;
  const sourceChainProvider = new ethers.providers.JsonRpcProvider(sourceRPC);
  const targetChainProviders = new Map<Chain, ethers.providers.Provider>();
  for (const key in NETWORKS[network]) {
    targetChainProviders.set(
      key as Chain,
      new ethers.providers.JsonRpcProvider(NETWORKS[network][key as Chain].rpc)
    );
  }

  // TODO: Convert this over to sdkv2
  // const info = await relayer.getWormholeRelayerInfo(chain, argv.tx, {
  //   environment: network,
  //   sourceChainProvider,
  //   targetChainProviders,
  // });
  // console.log(relayer.stringifyWormholeRelayerInfo(info));

  console.log("Not implemented");
};
