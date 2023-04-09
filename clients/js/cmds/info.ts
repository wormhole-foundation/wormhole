import yargs from "yargs";
import {
  CHAINS,
  assertChain,
  coalesceChainId,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { NETWORKS } from '../networks';

const chain_args = {
  describe: "Chain to query",
  type: "string",
  choices: Object.keys(CHAINS),
} as const;

exports.command = "info";
exports.desc = "Contract/chain/rpc information utilities";
exports.builder = (y: typeof yargs) => {
  return y
  .command("chain-id <chain>", "Print the wormhole chain ID integer associated with the specified chain name", (yargs) => {
    return yargs
      .positional("chain", chain_args)
  }, (argv) => {
    assertChain(argv["chain"]);
    console.log(coalesceChainId(argv["chain"]));
  })
  .command("rpc <network> <chain>", "Print RPC address", (yargs) => {
    return yargs
      .positional("network", {
        describe: "network",
        type: "string",
        choices: ["mainnet", "testnet", "devnet"],
      })
      .positional("chain", chain_args);
  }, (argv) => {
    assertChain(argv["chain"]);
    const network = argv.network.toUpperCase();
    if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
      throw Error(`Unknown network: ${network}`);
    }
    console.log(NETWORKS[network][argv["chain"]].rpc);
  })
};
