import fs from "fs";
import path from "path";
import yargs from "yargs";
import {
  CONTRACTS,
  GOVERNANCE_CHAIN,
  GOVERNANCE_EMITTER,
  INITIAL_GUARDIAN_DEVNET,
  RPC_OPTIONS,
} from "../../consts";
import { NETWORKS } from "../../networks";
import {
  getCreatedObjects,
  getPublishedPackageId,
  isSameType,
} from "../../sui";
import { logPublishedPackageId, logTransactionDigest } from "../../sui/log";
import { YargsAddCommandsFn } from "../Yargs";
import { deploy } from "./deploy";
import { initExampleApp, initTokenBridge, initWormhole } from "./init";

export const addSetupCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
    "setup-devnet",
    "Setup devnet by deploying and initializing core and token bridges and submitting chain registrations.",
    (yargs) => {
      return yargs
        .option("overwrite-ids", {
          alias: "o",
          describe: "Overwrite object IDs in the case that they've changed",
          required: false,
          default: false,
          type: "boolean",
        })
        .option("private-key", {
          alias: "k",
          describe: "Custom private key to sign txs",
          required: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS);
    },
    async (argv) => {
      const network = "DEVNET";
      const overwriteIds = argv["overwrite-ids"];
      const privateKey = argv["private-key"];
      const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;

      // Deploy & init core bridge
      console.log("[1/4] Deploying core bridge...");
      const coreBridgeDeployRes = await deploy(
        network,
        "wormhole",
        rpc,
        privateKey
      );
      logTransactionDigest(coreBridgeDeployRes);
      logPublishedPackageId(coreBridgeDeployRes);

      console.log("\n[2/4] Initializing core bridge...");
      const coreBridgePackageId = getPublishedPackageId(coreBridgeDeployRes);
      const coreBridgeInitRes = await initWormhole(
        network,
        coreBridgePackageId,
        INITIAL_GUARDIAN_DEVNET,
        GOVERNANCE_CHAIN,
        GOVERNANCE_EMITTER,
        rpc,
        privateKey
      );
      const coreBridgeStateObjectId = getCreatedObjects(coreBridgeInitRes).find(
        (e) => isSameType(e.type, `${coreBridgePackageId}::state::State`)
      ).objectId;
      logTransactionDigest(coreBridgeInitRes);
      console.log("Core bridge state object ID", coreBridgeStateObjectId);

      // Deploy & init token bridge
      console.log("\n[3/4] Deploying token bridge...");
      const tokenBridgeDeployRes = await deploy(
        network,
        "token_bridge",
        rpc,
        privateKey
      );
      logTransactionDigest(tokenBridgeDeployRes);
      logPublishedPackageId(tokenBridgeDeployRes);

      console.log("\n[4/4] Initializing token bridge...");
      const tokenBridgePackageId = getPublishedPackageId(tokenBridgeDeployRes);
      const tokenBridgeInitRes = await initTokenBridge(
        network,
        tokenBridgePackageId,
        coreBridgeStateObjectId,
        rpc,
        privateKey
      );
      const tokenBridgeStateObjectId = getCreatedObjects(
        tokenBridgeInitRes
      ).find((e) =>
        isSameType(e.type, `${tokenBridgePackageId}::state::State`)
      ).objectId;
      logTransactionDigest(tokenBridgeInitRes);
      console.log("Token bridge state object ID", tokenBridgeStateObjectId);

      // Overwrite object IDs if they've changed
      if (
        overwriteIds &&
        (coreBridgeStateObjectId !== CONTRACTS[network].sui.core ||
          tokenBridgeStateObjectId !== CONTRACTS[network].sui.token_bridge)
      ) {
        console.log("\nOverwriting object IDs...");
        const filepaths = [
          path.resolve(__dirname, `../../consts.ts`),
          path.resolve(__dirname, `../../../../sdk/js/sui/src/consts.ts`),
        ];
        for (const filepath of filepaths) {
          const text = fs.readFileSync(filepath, "utf8").toString();
          fs.writeFileSync(
            filepath,
            text
              .replace(CONTRACTS[network].sui.core, coreBridgeStateObjectId)
              .replace(
                CONTRACTS[network].sui.token_bridge,
                tokenBridgeStateObjectId
              )
          );
        }
      }

      // Deploy & init example app
      console.log("\n[+1/3] Deploying example app...");
      const exampleAppDeployRes = await deploy(
        network,
        "examples/core_messages",
        rpc,
        privateKey
      );
      logTransactionDigest(tokenBridgeDeployRes);
      logPublishedPackageId(tokenBridgeDeployRes);

      console.log("\n[+2/3] Initializing example app...");
      const exampleAppPackageId = getPublishedPackageId(exampleAppDeployRes);
      const exampleAppInitRes = await initExampleApp(
        network,
        exampleAppPackageId,
        coreBridgeStateObjectId,
        rpc,
        privateKey
      );
      const exampleAppStateObjectId = getCreatedObjects(exampleAppInitRes).find(
        (e) => isSameType(e.type, `${exampleAppPackageId}::sender::State`)
      ).objectId;
      logTransactionDigest(exampleAppInitRes);
      console.log("Example app state object ID", exampleAppStateObjectId);

      // Deploy example coins
      console.log("\n[+3/3] Deploying example coins...");
      const coinsDeployRes = await deploy(
        network,
        "examples/coins",
        rpc,
        privateKey
      );
      logTransactionDigest(coinsDeployRes);
      logPublishedPackageId(coinsDeployRes);

      // Print publish message command for convenience
      let publishCommand = `\nPublish message command: worm sui publish-example-message -n devnet -p "${exampleAppPackageId}" -s "${exampleAppStateObjectId}" -w "${coreBridgeStateObjectId}" -m "hello"`;
      if (argv.rpc) publishCommand += ` -r "${argv.rpc}"`;
      if (privateKey) publishCommand += ` -k "${privateKey}"`;
      console.log(publishCommand);
    }
  );
