import {
  normalizeSuiAddress,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import yargs from "yargs";
import {
  executeTransactionBlock,
  getProvider,
  getSigner,
  logTransactionDigest,
  logTransactionSender,
  setMaxGasBudgetDevnet,
} from "../../chains/sui";
import { NETWORK_OPTIONS, NETWORKS, RPC_OPTIONS } from "../../consts";
import { assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addPublishMessageCommands: YargsAddCommandsFn = (
  y: typeof yargs
) =>
  y.command(
    "publish-example-message",
    "Publish message from example app via core bridge",
    (yargs) =>
      yargs
        .option("network", NETWORK_OPTIONS)
        .option("package-id", {
          alias: "p",
          describe: "Package ID/module address",
          demandOption: true,
          type: "string",
        })
        .option("state", {
          alias: "s",
          describe: "Core messages app state object ID",
          demandOption: true,
          type: "string",
        })
        .option("wormhole-state", {
          alias: "w",
          describe: "Wormhole state object ID",
          demandOption: true,
          type: "string",
        })
        .option("message", {
          alias: "m",
          describe: "Message payload",
          demandOption: true,
          type: "string",
        })
        .option("private-key", {
          alias: "k",
          describe: "Custom private key to sign txs",
          demandOption: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS),
    async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const packageId = argv["package-id"];
      const stateObjectId = argv.state;
      const wormholeStateObjectId = argv["wormhole-state"];
      const message = argv.message;
      const privateKey = argv["private-key"];
      const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;

      const provider = getProvider(network, rpc);
      const signer = getSigner(provider, network, privateKey);

      // Publish message
      const tx = new TransactionBlock();
      setMaxGasBudgetDevnet(network, tx);
      tx.moveCall({
        target: `${packageId}::sender::send_message_entry`,
        arguments: [
          tx.object(stateObjectId),
          tx.object(wormholeStateObjectId),
          tx.pure(message),
          tx.object(SUI_CLOCK_OBJECT_ID),
        ],
      });
      const res = await executeTransactionBlock(signer, tx);

      // Hacky way to grab event since we don't require package ID of the
      // core bridge as input. Doesn't really matter since this is a test
      // command.
      const event = res.events?.find(
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
        payload: Buffer.from(event.parsedJson?.payload).toString(),
        emitter: Buffer.from(event.parsedJson?.sender).toString("hex"),
        sequence: event.parsedJson?.sequence,
        nonce: event.parsedJson?.nonce,
      });
    }
  );
