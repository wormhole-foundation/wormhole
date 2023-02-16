import { JsonRpcProvider } from "@mysten/sui.js";
import yargs from "yargs";
import { config } from "../config";
import { NETWORK_OPTIONS, RPC_OPTIONS } from "../consts";
import { NETWORKS } from "../networks";
import { callEntryFunc, loadSigner, publishPackage } from "../sui";
import { assertNetwork, checkBinary } from "../utils";

/*
  Loop through a list of Sui objects and look for the DeployerCapability that should
  have been granted upon publication of the package with package_id

  The objects is in the format returned by a json-rpc call "provider.getObjectsOwnedByAddress"
*/
function findDeployerCapability(
  packageId: string,
  moduleName: string,
  objects: any[]
): string | null {
  const type = `${packageId}::${moduleName}::DeployerCapability`;
  return (
    objects.find((o) => o.type.toLowerCase() === type.toLowerCase())
      ?.objectId ?? null
  );
}

exports.command = "sui";
exports.desc = "Sui utilities ";
exports.builder = function (y: typeof yargs) {
  return y
    .command(
      "deploy <package-dir>",
      "Deploy a Sui package",
      (yargs) => {
        return yargs
          .positional("package-dir", {
            type: "string",
          })
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        checkBinary("sui", "sui");

        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageDir = argv["package-dir"];
        const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;

        console.log("package: ", packageDir);
        console.log("network: ", network);
        console.log("rpc: ", rpc);

        // TODO(aki): should user pass in entire path to package?
        await publishPackage(
          network,
          rpc,
          `${config.wormholeDir}/sui/${packageDir}`
        );
      }
    )
    .command(
      "get-owned-objects",
      "Get owned objects by owner",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("owner", {
            alias: "o",
            describe: "Owner address",
            required: true,
            type: "string",
          });
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
        const provider = new JsonRpcProvider(rpc);
        const owner = argv.owner;
        const objects = await provider.getObjectsOwnedByAddress(owner);
        console.log("network: ", network);
        console.log("owner: ", owner);
        console.log("objects: ", JSON.stringify(objects));
      }
    )
    .command(
      "init-token-bridge",
      "Init token bridge contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("package-id", {
            describe: "Package/module ID",
            required: true,
            type: "string",
          })
          .option("worm-state", {
            alias: "e",
            describe: "Wormhole core bridge state object ID",
            required: true,
            type: "string",
          });
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
        const packageId = argv["package-id"];
        const provider = new JsonRpcProvider(rpc);
        const signer = loadSigner(network, rpc);
        const owner = await signer.getAddress();
        console.log("owner: ", owner);
        const objects = await provider.getObjectsOwnedByAddress(owner);
        const wormState = argv["worm-state"];
        const deployer = findDeployerCapability(
          packageId,
          "bridge_state",
          objects
        );

        console.log("network: ", network);
        console.log("rpc: ", rpc);
        console.log("package id: ", packageId);
        console.log("deployer object id: ", deployer);
        console.log("wormhole state object id: ", wormState);

        await callEntryFunc(
          network,
          rpc,
          String(packageId),
          "bridge_state",
          "init_and_share_state",
          [],
          [deployer, wormState]
        );
      }
    )
    .command(
      "init-wormhole",
      "Init wormhole core contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("package-id", {
            describe: "Package/module ID",
            required: true,
            type: "string",
          })
          .option("chain-id", {
            alias: "ci",
            describe: "Chain ID",
            default: "22",
            required: false,
            type: "string",
          })
          .option("governance-chain-id", {
            alias: "gci",
            describe: "Governance chain ID",
            default: "1", // default is chain ID of Solana
            type: "string",
            required: false,
          })
          .option("governance-contract", {
            alias: "gc",
            describe: "Governance contract",
            type: "string",
            default:
              "0000000000000000000000000000000000000000000000000000000000000004",
            required: false,
          })
          .option("initial-guardian", {
            alias: "ig",
            required: true,
            describe: "Initial guardian public keys",
            type: "string",
          });
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network]["sui"].rpc;
        const packageId = argv["package-id"];
        const chainId = argv["chain-id"];
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-contract"];
        const initialGuardian = argv["initial-guardian"];
        const provider = new JsonRpcProvider(rpc);
        const signer = loadSigner(network, rpc);
        const owner = await signer.getAddress();
        const objects = await provider.getObjectsOwnedByAddress(owner);
        const deployer = findDeployerCapability(packageId, "state", objects);
        if (typeof deployer == "undefined") {
          throw new Error(
            "Wormhole core bridge cannot be initialized because deployer capability cannot be found. Is the package published?"
          );
        }

        console.log("network: ", network);
        console.log("rpc: ", rpc);
        console.log("package id: ", packageId);
        console.log("deployer object id: ", deployer);
        console.log("chain-id: ", chainId);
        console.log("governance-chain-id: ", governanceChainId);
        console.log("governance-contract: ", governanceContract);
        console.log("initial-guardian: ", initialGuardian);

        await callEntryFunc(
          network,
          rpc,
          String(packageId),
          "state",
          "init_and_share_state",
          [],
          [
            deployer,
            chainId,
            governanceChainId,
            [...Buffer.from(governanceContract, "hex")],
            [[...Buffer.from(initialGuardian, "hex")]],
          ]
        );
      }
    )
    .strict()
    .demandCommand();
};
