import yargs from "yargs";
import { config } from "../config";
import {
  NAMED_ADDRESSES_OPTIONS,
  NETWORK_OPTIONS,
  RPC_OPTIONS,
} from "../consts";
import { NETWORKS } from "../networks";
import {
  getOwnedObjectId,
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
          packageDir.startsWith("/") // Allow absolute paths, otherwise assume relative to sui directory
            ? packageDir
            : `${config.wormholeDir}/sui/${packageDir}`,
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

        // todo(aki): handle pagination
        const provider = getProvider(network, rpc);
        const objects = await provider.getOwnedObjects({ owner });

        console.log("Network: ", network);
        console.log("Owner:   ", owner);
        console.log("Objects: ", JSON.stringify(objects, null, 2));
      }
    )
    .command(
      "init-token-bridge",
      "Initialize token bridge contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            required: true,
            type: "string",
          });
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const packageId = argv["package-id"];
        const wormholeStateObjectId = argv["wormhole-state"];

        const provider = getProvider(network, rpc);
        const signer = getSigner(provider, network);
        const owner = await signer.getAddress();
        console.log("Owner:                    ", owner);
        const deployerCapObjectId = await getOwnedObjectId(
          provider,
          owner,
          packageId,
          "state",
          "DeployerCap"
        );

        console.log("Network:                  ", network);
        console.log("Package ID:               ", packageId);
        console.log("Deployer object ID:       ", deployerCapObjectId);
        console.log("Wormhole state object ID: ", wormholeStateObjectId);

        if (!deployerCapObjectId) {
          throw new Error(
            `Token bridge cannot be initialized because deployer capability cannot be found under ${owner}. Is the package published?`
          );
        }

        // const effects: TransactionEffects =
        // await executeTransactionBlock(
        //   provider,
        //   network,
        //   packageId,
        //   "state",
        //   "init_and_share_state",
        //   [],
        //   [deployerCapObjectId, wormholeStateObjectId]
        // );

        // console.log(
        //   "Token bridge state object ID: ",
        //   effects["created"].find(
        //     (o) => typeof o.owner === "object" && "Shared" in o.owner
        //   ).reference.objectId
        // );
      }
    )
    .command(
      "init-wormhole",
      "Initialize wormhole core contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
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
        const deployerCapObjectId = await getOwnedObjectId(
          provider,
          owner,
          packageId,
          "setup",
          "DeployerCap"
        );
        const upgradeCapObjectId = await getOwnedObjectId(
          provider,
          owner,
          packageId,
          "dummy_sui_package",
          "UpgradeCap"
        );

        console.log("Network:               ", network);
        console.log("RPC:                   ", rpc);
        console.log("Package ID:            ", packageId);
        console.log("Deployer cap object ID:", deployerCapObjectId);
        console.log("Upgrade cap object ID: ", upgradeCapObjectId);
        console.log("Chain ID:              ", chainId);
        console.log("Governance chain ID:   ", governanceChainId);
        console.log("Governance contract:   ", governanceContract);
        console.log("Initial guardian:      ", initialGuardian);

        if (!deployerCapObjectId) {
          throw new Error(
            `Wormhole cannot be initialized because deployer capability cannot be found under ${owner}. Is the package published?`
          );
        }

        if (!upgradeCapObjectId) {
          throw new Error(
            `Wormhole cannot be initialized because upgrade capability cannot be found under ${owner}. Is the package published?`
          );
        }

        // const effects: TransactionEffects =
        // await executeTransactionBlock(
        //   provider,
        //   network,
        //   packageId,
        //   "setup",
        //   "init_and_share_state",
        //   [],
        //   [
        //     deployerCapObjectId,
        //     upgradeCapObjectId,
        //     governanceChainId,
        //     [...Buffer.from(governanceContract, "hex")],
        //     [[...Buffer.from(initialGuardian, "hex")]],
        //     365, // Guardian set TTL in epochs
        //     "0", // Message fee
        //   ]
        // );

        // console.log(
        //   "Wormhole state object ID: ",
        //   effects["created"].find(
        //     (o) => typeof o.owner === "object" && "Shared" in o.owner
        //   ).reference.objectId
        // );
      }
    )
    .command(
      "publish-message",
      "Publish message from example app via core bridge",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("state", {
            alias: "s",
            describe: "Example app state object ID",
            required: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            required: true,
            type: "string",
          })
          .option("message", {
            alias: "m",
            describe: "Message payload",
            required: true,
            type: "string",
          });
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const packageId = argv["package-id"];
        const stateObjectId = argv["state"];
        const wormholeStateObjectId = argv["wormhole-state"];
        const message = argv["message"];

        const provider = getProvider(network, rpc);
        const signer = getSigner(provider, network);
        const owner = await signer.getAddress();

        // WH message fee is 0 for devnet deployment
        // TODO(aki): Read from on-chain state since it can technically change
        const feeAmount = BigInt(0);

        // Get fee
        // const feeCoins = (
        //   await provider.selectCoinsWithBalanceGreaterThanOrEqual(
        //     owner,
        //     feeAmount,
        //     "0x2::sui::SUI"
        //   )
        // ).find((c) => c.status === "Exists");
        // if (!feeCoins) {
        //   throw new Error(
        //     `Cannot find SUI coins owned by ${owner} with sufficient balance`
        //   );
        // }

        // const effects: TransactionEffects =
        // await executeTransactionBlock(
        //   provider,
        //   network,
        //   packageId,
        //   "sender",
        //   "send_message_entry",
        //   [],
        //   [stateObjectId, wormholeStateObjectId, message, getObjectId(feeCoins)]
        // );

        // const event = effects.events.find((e) => "moveEvent" in e) as
        //   | PublishMessageEvent
        //   | undefined;
        // if (!event) {
        //   throw new Error("Publish failed");
        // }

        // console.log("Publish message succeeded:", {
        //   sender: event.moveEvent.sender,
        //   type: event.moveEvent.type,
        //   payload: Buffer.from(event.moveEvent.fields.payload).toString(),
        //   emitter: Buffer.from(event.moveEvent.fields.sender).toString("hex"),
        //   sequence: event.moveEvent.fields.sequence,
        // });
      }
    )
    .strict()
    .demandCommand();
};

type PublishMessageEvent = {
  moveEvent: {
    type: string;
    fields: {
      consistency_level: number;
      nonce: number; // u32
      payload: Uint8Array;
      sender: Uint8Array;
      sequence: string; // u64
    };
    sender: string;
    packageId: string;
    transactionModule: string;
    bcs: string;
  };
};
