import yargs from "yargs";
import { callEntryFunc, publishPackage} from "../sui";
import { spawnSync } from 'child_process';
import { NETWORKS } from "../networks";
import { config } from '../config';

type Network = "MAINNET" | "TESTNET" | "DEVNET"

function assertNetwork(n: string): asserts n is Network {
  if (
    n !== "MAINNET" &&
      n !== "TESTNET" &&
      n !== "DEVNET"
  ) {
    throw Error(`Unknown network: ${n}`);
  }
}

const network_options = {
  alias: "n",
  describe: "network",
  type: "string",
  choices: ["mainnet", "testnet", "devnet"],
  required: true,
} as const;

const rpc_description = {
  alias: "r",
  describe: "override default rpc endpoint url",
  type: "string",
  required: false,
} as const;

const dir = `${config.wormholeDir}/sui`;

exports.command = 'sui';
exports.desc = 'Sui utilities ';
exports.builder = function(y: typeof yargs) {
  return y
    .command("init-wormhole", "Publish Wormhole core contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      console.log("network: ", network)
      console.log("rpc: ", rpc)
      await publishPackage(network, rpc, `${dir}/wormhole`);
    })
    .command("init-coin", "Publish coin contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      console.log("network: ", network)
      console.log("rpc: ", rpc)
      await publishPackage(network, rpc, `${dir}/coin`);
    })
    .strict().demandCommand();
}

