import yargs, { string } from "yargs";
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
    .command("publish-wormhole", "Publish Wormhole core contract", (yargs) => {
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
    .command("publish-tokenbridge", "Publish Wormhole token bridge contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      console.log("network: ", network)
      console.log("rpc: ", rpc)
      await publishPackage(network, rpc, `${dir}/tokenbridge`);
    })
    .command("init-wormhole", "Init wormhole core contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("deployer", {
          alias: "d",
          describe: "Deployer capability object ID",
          required: true,
          type: "string",
        })
        .option("chain-id", {
          alias: "ci",
          describe: "Chain ID",
          default: 22,
          required: false,
          type: "number"
        })
        .option("governance-chain-id", {
          alias: "gci",
          describe: "Governance chain ID",
          default: 1,
          type: "number",
          required: false
        })
        .option("governance-contract", {
          alias: "gc",
          describe: "Governance contract",
          type: "string",
          default: "0x0000000000000000000000000000000000000000000000000000000000000004",
          required: false
        })
        .option("initial-guardian", {
          alias: "ig",
          required: true,
          describe: "Initial guardian public keys)",
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      const deployer = argv.deployer;
      const chain_id = argv["chain-id"];
      const governance_chain_id = argv["governance-chain-id"];
      const governance_contract = argv["governance-contract"];
      const initial_guardian = argv["initial-guardian"];
      console.log("network: ", network)
      console.log("rpc: ", rpc)
      console.log("deployer: ", deployer)
      console.log("chain-id: ", chain_id)
      console.log("governance-chain-id: ", governance_chain_id)
      console.log("governance-contract: ", governance_contract)
      console.log("initial-guardian: ", initial_guardian)
      await callEntryFunc(
        network,
        rpc,
        "packageObjectId",
        "state",
        "init_and_share_state",
        [],
        [
          deployer,
          chain_id,
          governance_chain_id,
          governance_contract,
          [initial_guardian]
        ],
    )
      //await publishPackage(network, rpc, `${dir}/tokenbridge`);
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

