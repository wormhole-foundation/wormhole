import { TransactionBlock } from "@mysten/sui.js";
import yargs from "yargs";
import { config } from "../config";
import {
  GOVERNANCE_CHAIN,
  GOVERNANCE_EMITTER,
  NETWORK_OPTIONS,
  RPC_OPTIONS,
} from "../consts";
import { NETWORKS } from "../networks";
import {
  executeTransactionBlock,
  getOwnedObjectId,
  getProvider,
  getSigner,
  isSuiCreateEvent,
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
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        checkBinary("sui", "sui");

        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageDir = argv["package-dir"];
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const provider = getProvider(network, rpc);

        console.log("Package", packageDir);
        console.log("RPC", rpc);

        await publishPackage(
          provider,
          network,
          packageDir.startsWith("/") // Allow absolute paths, otherwise assume relative to sui directory
            ? packageDir
            : `${config.wormholeDir}/sui/${packageDir}`
        );
      }
    )
    .command(
      "get-owned-objects",
      "Get owned objects by owner",
      (yargs) => {
        return yargs
          .positional("owner", {
            describe: "Owner address",
            type: "string",
          })
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const owner = argv.owner;

        // todo(aki): handle pagination
        const provider = getProvider(network, rpc);
        const objects = await provider.getOwnedObjects({ owner });

        console.log("Network", network);
        console.log("Owner", owner);
        console.log("Objects", JSON.stringify(objects, null, 2));
      }
    )
    .command(
      "init-token-bridge",
      "Initialize token bridge contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
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
          })
          .option("rpc", RPC_OPTIONS);
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
        const deployerCapObjectId = await getOwnedObjectId(
          provider,
          owner,
          packageId,
          "state",
          "DeployerCap"
        );

        console.log("Owner", owner);
        console.log("Network", network);
        console.log("Package ID", packageId);
        console.log("Deployer object ID", deployerCapObjectId);
        console.log("Wormhole state object ID", wormholeStateObjectId);

        if (!deployerCapObjectId) {
          throw new Error(
            `Token bridge cannot be initialized because deployer capability cannot be found under ${owner}. Is the package published?`
          );
        }

        const transactionBlock = new TransactionBlock();
        transactionBlock.moveCall({
          target: `${packageId}::state::init_and_share_state`,
          arguments: [
            transactionBlock.object(deployerCapObjectId),
            transactionBlock.object(wormholeStateObjectId),
          ],
        });
        const res = await executeTransactionBlock(
          provider,
          network,
          transactionBlock
        );
        console.log(
          "Token bridge state object ID",
          res.objectChanges
            .filter(isSuiCreateEvent)
            .find((e) => e.objectType === `${packageId}::state::State`).objectId
        );
      }
    )
    .command(
      "init-wormhole",
      "Initialize wormhole core contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("initial-guardian", {
            alias: "i",
            required: true,
            describe: "Initial guardian public keys",
            type: "string",
          })
          .option("governance-chain-id", {
            alias: "c",
            describe: "Governance chain ID",
            default: GOVERNANCE_CHAIN,
            type: "string",
            required: false,
          })
          .option("governance-contract-address", {
            alias: "a",
            describe: "Governance contract address",
            type: "string",
            default: GOVERNANCE_EMITTER,
            required: false,
          })
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const packageId = argv["package-id"];
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-contract-address"];
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
          "0x2",
          "package",
          "UpgradeCap"
        );

        console.log("Network", network);
        console.log("RPC", rpc);
        console.log("Package ID", packageId);
        console.log("Deployer cap object ID", deployerCapObjectId);
        console.log("Upgrade cap object ID", upgradeCapObjectId);
        console.log("Governance chain ID", governanceChainId);
        console.log("Governance contract", governanceContract);
        console.log("Initial guardian", initialGuardian);

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

        const transactionBlock = new TransactionBlock();
        transactionBlock.moveCall({
          target: `${packageId}::setup::init_and_share_state`,
          arguments: [
            transactionBlock.object(deployerCapObjectId),
            transactionBlock.object(upgradeCapObjectId),
            transactionBlock.pure(governanceChainId),
            transactionBlock.pure([...Buffer.from(governanceContract, "hex")]),
            transactionBlock.pure([[...Buffer.from(initialGuardian, "hex")]]),
            transactionBlock.pure(365), // Guardian set TTL in epochs
            transactionBlock.pure("0"), // Message fee
          ],
        });
        const res = await executeTransactionBlock(
          provider,
          network,
          transactionBlock
        );
        console.log(
          "Wormhole state object ID",
          res.objectChanges
            .filter(isSuiCreateEvent)
            .find((e) => e.objectType === `${packageId}::state::State`).objectId
        );
      }
    )
    .command(
      "publish-example-message",
      "Publish message from example app via core bridge",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("state", {
            alias: "s",
            describe: "Core messages app state object ID",
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
          })
          .option("rpc", RPC_OPTIONS);
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

        // WH message fee is 0 for devnet
        // TODO(aki): Read from on-chain state since it can technically change
        const feeAmount = BigInt(0);

        // Get fee
        const transactionBlock = new TransactionBlock();
        const [feeCoin] = transactionBlock.splitCoins(transactionBlock.gas, [
          transactionBlock.pure(feeAmount),
        ]);

        // Publish message
        transactionBlock.moveCall({
          target: `${packageId}::sender::send_message_entry`,
          arguments: [
            transactionBlock.object(stateObjectId),
            transactionBlock.object(wormholeStateObjectId),
            transactionBlock.pure(message),
            feeCoin,
          ],
        });
        const res = await executeTransactionBlock(
          provider,
          network,
          transactionBlock
        );

        // Hacky way to grab event since we don't require package ID of the
        // core bridge as input. Doesn't really matter since this is a test
        // command.
        const event = res.events.find(
          (e) =>
            e.packageId === packageId &&
            e.type.includes("publish_message::WormholeMessage")
        );
        if (!event) {
          throw new Error("Publish failed");
        }

        console.log("Publish message succeeded:", {
          sender: event.sender,
          type: event.type,
          payload: Buffer.from(event.parsedJson.payload).toString(),
          emitter: Buffer.from(event.parsedJson.sender).toString("hex"),
          sequence: event.parsedJson.sequence,
          nonce: event.parsedJson.nonce,
        });
      }
    )
    .strict()
    .demandCommand();
};
