import {
  normalizeSuiAddress,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import yargs from "yargs";
import { NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { executeTransactionBlock, getProvider, getSigner } from "../../sui";
import { logTransactionDigest, logTransactionSender } from "../../sui/log";
import { assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addPublishMessageCommands: YargsAddCommandsFn = (
  y: typeof yargs
) =>
  y.command(
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
        .option("private-key", {
          alias: "k",
          describe: "Custom private key to sign txs",
          required: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS);
    },
    async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const packageId = argv["package-id"];
      const stateObjectId = argv["state"];
      const wormholeStateObjectId = argv["wormhole-state"];
      const message = argv["message"];
      const privateKey = argv["private-key"];
      const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;

      const provider = getProvider(network, rpc);
      const signer = getSigner(provider, network, privateKey);

      // Publish message
      const transactionBlock = new TransactionBlock();
      transactionBlock.moveCall({
        target: `${packageId}::sender::send_message_entry`,
        arguments: [
          transactionBlock.object(stateObjectId),
          transactionBlock.object(wormholeStateObjectId),
          transactionBlock.pure(message),
          transactionBlock.object(SUI_CLOCK_OBJECT_ID),
        ],
      });
      const res = await executeTransactionBlock(signer, transactionBlock);

      // Hacky way to grab event since we don't require package ID of the
      // core bridge as input. Doesn't really matter since this is a test
      // command.
      const event = res.events.find(
        (e) =>
          normalizeSuiAddress(e.packageId) === normalizeSuiAddress(packageId) &&
          e.type.includes("publish_message::WormholeMessage")
      );
      if (!event) {
        throw new Error(
          "Couldn't find publish event. Events: " +
            JSON.stringify(res.events, null, 2)
        );
      }

      logTransactionDigest(res);
      logTransactionSender(res);
      console.log("Publish message succeeded:", {
        sender: event.sender,
        type: event.type,
        payload: Buffer.from(event.parsedJson.payload).toString(),
        emitter: Buffer.from(event.parsedJson.sender).toString("hex"),
        sequence: event.parsedJson.sequence,
        nonce: event.parsedJson.nonce,
      });
    }
  );
