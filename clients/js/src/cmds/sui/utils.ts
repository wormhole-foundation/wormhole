import { PaginatedObjectsResponse } from "@mysten/sui.js";
import yargs from "yargs";
import { getPackageId, getProvider } from "../../chains/sui";
import { NETWORKS, NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addUtilsCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y
    .command(
      "objects <owner>",
      "Get owned objects by owner",
      (yargs) =>
        yargs
          .positional("owner", {
            describe: "Owner address",
            type: "string",
            demandOption: true,
          })
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const owner = argv.owner;

        const provider = getProvider(network, rpc);
        const objects: PaginatedObjectsResponse["data"] = [];

        let cursor: PaginatedObjectsResponse["nextCursor"] | undefined =
          undefined;
        while (true) {
          const res: PaginatedObjectsResponse = await provider.getOwnedObjects({
            owner,
            cursor,
          });
          objects.push(...res.data);
          if (res.hasNextPage) {
            cursor = res.nextCursor;
          } else {
            break;
          }
        }

        console.log("Network", network);
        console.log("Owner", owner);
        console.log("Objects", JSON.stringify(objects, null, 2));
      }
    )
    .command(
      "package-id <state-object-id>",
      "Get package ID from State object ID",
      (yargs) =>
        yargs
          .positional("state-object-id", {
            describe: "Object ID of State object",
            type: "string",
            demandOption: true,
          })
          .option("network", NETWORK_OPTIONS)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const provider = getProvider(network, rpc);
        console.log(await getPackageId(provider, argv["state-object-id"]));
      }
    )
    // This command is useful for debugging, especially when the Sui explorer
    // goes down :)
    .command(
      "tx <transaction-digest>",
      "Get transaction details",
      (yargs) =>
        yargs
          .positional("transaction-digest", {
            describe: "Digest of transaction to fetch",
            type: "string",
            demandOption: true,
          })
          .option("network", {
            alias: "n",
            describe: "Network",
            choices: ["mainnet", "testnet", "devnet"],
            default: "devnet",
            demandOption: false,
          } as const)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;
        const provider = getProvider(network, rpc);
        console.log(
          JSON.stringify(
            await provider.getTransactionBlock({
              digest: argv["transaction-digest"],
              options: {
                showInput: true,
                showEffects: true,
                showEvents: true,
                showObjectChanges: true,
              },
            }),
            null,
            2
          )
        );
      }
    );
