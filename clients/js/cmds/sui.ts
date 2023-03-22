import { TransactionEffects } from "@mysten/sui.js";
import yargs from "yargs";
import { config } from "../config";
import {
  NAMED_ADDRESSES_OPTIONS,
  NETWORK_OPTIONS,
  RPC_OPTIONS,
} from "../consts";
import { NETWORKS } from "../networks";
import {
  executeEntry,
  getObjectFromOwner,
  getProvider,
  getSigner,
  publishPackage,
} from "../sui";
import { assertNetwork, checkBinary } from "../utils";

exports.command = "sui";
exports.desc = "Sui utilities";
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
          .option("rpc", RPC_OPTIONS)
          .option("named-addresses", NAMED_ADDRESSES_OPTIONS);
      },
      async (argv) => {
        checkBinary("sui", "sui");

        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageDir = argv["package-dir"];
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const provider = getProvider(network, rpc);
        const namedAddresses = Object.fromEntries(
          (argv["named-addresses"] || "")
            .split(",")
            .map((str) => str.trim().split("="))
        );

        console.log("Package:         ", packageDir);
        console.log("RPC:             ", rpc);

        await publishPackage(
          provider,
          network,
          `${config.wormholeDir}/sui/${packageDir}`,
          namedAddresses
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
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const owner = argv.owner;

        const provider = getProvider(network, rpc);
        const objects = await provider.getObjectsOwnedByAddress(owner);

        console.log("Network: ", network);
        console.log("Owner:   ", owner);
        console.log("Objects: ", JSON.stringify(objects));
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
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const packageId = argv["package-id"];
        const wormState = argv["worm-state"];

        const provider = getProvider(network, rpc);
        const signer = getSigner(provider, network);
        const owner = await signer.getAddress();
        console.log("Owner:                    ", owner);
        const deployer = await getObjectFromOwner(
          provider,
          owner,
          packageId,
          "state",
          "DeployerCapability"
        );

        console.log("Network:                  ", network);
        console.log("Package ID:               ", packageId);
        console.log("Deployer object ID:       ", deployer);
        console.log("Wormhole state object ID: ", wormState);

        const effects: TransactionEffects = await executeEntry(
          provider,
          network,
          packageId,
          "state",
          "init_and_share_state",
          [],
          [deployer, wormState]
        );

        console.log(
          "Token bridge state object ID: ",
          effects["created"].find(
            (o) => typeof o.owner === "object" && "Shared" in o.owner
          ).reference.objectId
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
            default: "1", // Default is chain ID of Solana
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
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const packageId = argv["package-id"];
        const chainId = argv["chain-id"];
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-contract"];
        const initialGuardian = argv["initial-guardian"];

        const provider = getProvider(network, rpc);
        const signer = getSigner(provider, network);
        const owner = await signer.getAddress();
        const deployer = await getObjectFromOwner(
          provider,
          owner,
          packageId,
          "setup",
          "DeployerCapability"
        );
        if (typeof deployer == "undefined") {
          throw new Error(
            "Wormhole core bridge cannot be initialized because deployer capability cannot be found. Is the package published?"
          );
        }

        console.log("Network:             ", network);
        console.log("RPC:                 ", rpc);
        console.log("Package ID:          ", packageId);
        console.log("Deployer object ID:  ", deployer);
        console.log("Chain ID:            ", chainId);
        console.log("Governance chain ID: ", governanceChainId);
        console.log("Governance contract: ", governanceContract);
        console.log("Initial guardian:    ", initialGuardian);

        const effects: TransactionEffects = await executeEntry(
          provider,
          network,
          packageId,
          "setup",
          "init_and_share_state",
          [],
          [
            deployer,
            governanceChainId,
            [...Buffer.from(governanceContract, "hex")],
            [[...Buffer.from(initialGuardian, "hex")]],
            15, // Guardian set TTL in epochs
            "0", // Message fee
          ]
        );

        console.log(
          "Wormhole state object ID: ",
          effects["created"].find(
            (o) => typeof o.owner === "object" && "Shared" in o.owner
          ).reference.objectId
        );
      }
    )
    .strict()
    .demandCommand();
};
