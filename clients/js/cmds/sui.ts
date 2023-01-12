import yargs, { string } from "yargs";
import { callEntryFunc, publishPackage, loadSigner} from "../sui";
import { JsonRpcProvider } from '@mysten/sui.js';
import { spawnSync } from 'child_process';
import { NETWORKS } from "../networks";
import { config } from '../config';
import {Network, assertNetwork} from "../utils";

/*
  Loop through a list of Sui objects and look for the DeployerCapability that should
  have been granted upon publication of the package with package_id

  The objects is in the format returned by a json-rpc call "provider.getObjectsOwnedByAddress"
*/
function findDeployerCapability(packageId: string, moduleName: string, objects: any[]): string | null {
  const type = `${packageId}::${moduleName}::DeployerCapability`;
  return objects.find(o => o.type.toLowerCase() === type.toLowerCase())?.objectId ?? null;
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
    .command("get-owned-objects", "Get owned objects by owner", (yargs)=>{
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("owner", {
          alias: "o",
          describe: "Owner address",
          required: true,
          type: "string",
        })
    }, async (argv)=>{
      const network = argv.network.toUpperCase()
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      const provider = new JsonRpcProvider(rpc);
      const owner = argv.owner;
      const objects = await provider.getObjectsOwnedByAddress(
        owner
      );
      console.log("network: ", network)
      console.log("owner: ", owner)
      console.log("objects: ", JSON.stringify(objects))
    })
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
    .command("publish-token-bridge", "Publish Wormhole token bridge contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      console.log("network: ", network)
      console.log("rpc: ", rpc)
      await publishPackage(network, rpc, `${dir}/token_bridge`);
    })
    .command("init-wormhole", "Init wormhole core contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("package-id", {
          describe: "Package/module ID",
          required: true,
          type: "string"
        })
        .option("chain-id", {
          alias: "ci",
          describe: "Chain ID",
          default: "22",
          required: false,
          type: "string"
        })
        .option("governance-chain-id", {
          alias: "gci",
          describe: "Governance chain ID",
          default: "1", // default is chain ID of Solana
          type: "string",
          required: false
        })
        .option("governance-contract", {
          alias: "gc",
          describe: "Governance contract",
          type: "string",
          default: "0000000000000000000000000000000000000000000000000000000000000004",
          required: false
        })
        .option("initial-guardian", {
          alias: "ig",
          required: true,
          describe: "Initial guardian public keys",
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      const package_id = argv["package-id"]
      const chain_id = argv["chain-id"];
      const governance_chain_id = argv["governance-chain-id"];
      const governance_contract = argv["governance-contract"];
      const initial_guardian = argv["initial-guardian"];
      const provider = new JsonRpcProvider(rpc);
      const signer = loadSigner(network, rpc);
      const owner = await signer.getAddress()
      const objects = await provider.getObjectsOwnedByAddress(
        owner
      );
      const deployer = findDeployerCapability(package_id, "state", objects)
      if (typeof deployer == 'undefined'){
        throw new Error('Wormhole core bridge cannot be initialized because deployer capability cannot be found. Is the package published?');
      }

      console.log("network: ", network)
      console.log("rpc: ", rpc)
      console.log("package id: ", package_id)
      console.log("deployer object id: ", deployer)
      console.log("chain-id: ", chain_id)
      console.log("governance-chain-id: ", governance_chain_id)
      console.log("governance-contract: ", governance_contract)
      console.log("initial-guardian: ", initial_guardian)

      await callEntryFunc(
        network,
        rpc,
        String(package_id),
        "state",
        "init_and_share_state",
        [],
        [
          deployer,
          chain_id,
          governance_chain_id,
          [...Buffer.from(governance_contract, "hex")],
          [[...Buffer.from(initial_guardian, "hex")]]
        ],
    )
    })
    .command("init-token-bridge", "Init token bridge contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("package-id", {
          describe: "Package/module ID",
          required: true,
          type: "string"
        })
        .option("worm-state", {
          alias: "e",
          describe: "Wormhole core bridge state object ID",
          required: true,
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
      const package_id = argv["package-id"]
      const provider = new JsonRpcProvider(rpc);
      const signer = loadSigner(network, rpc);
      const owner = await signer.getAddress()
      console.log("owner: ", owner)
      const objects = await provider.getObjectsOwnedByAddress(
        owner
      );
      const worm_state = argv["worm-state"]
      const deployer = findDeployerCapability(package_id, "bridge_state", objects)

      console.log("network: ", network)
      console.log("rpc: ", rpc)
      console.log("package id: ", package_id)
      console.log("deployer object id: ", deployer)
      console.log("wormhole state object id: ", worm_state)

      await callEntryFunc(
        network,
        rpc,
        String(package_id),
        "bridge_state",
        "init_and_share_state",
        [],
        [
          deployer,
          worm_state
        ],
    )
    })
    .command("publish-coin", "Publish coin contract", (yargs) => {
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
