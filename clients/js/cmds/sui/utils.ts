import yargs from "yargs";
import { NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { getProvider } from "../../sui";
import { assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addUtilsCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
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
  );
