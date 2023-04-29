import yargs from "yargs";
import { CONTRACTS, NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { getProvider } from "../../sui";
import { getCoinBuildOutputManual } from "../../sui/buildCoin";
import { assertNetwork, checkBinary } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addBuildCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
    "build-coin",
    "Build wrapped coin and dump bytecode",
    (yargs) => {
      return yargs
        .option("decimals", {
          alias: "d",
          describe: "Decimals of asset",
          required: true,
          type: "number",
        })
        .option("network", NETWORK_OPTIONS)
        .option("wormhole-state", {
          alias: "w",
          describe: "Wormhole state object ID",
          required: false,
          type: "string",
        })
        .option("token-bridge-state", {
          alias: "t",
          describe: "Token bridge state object ID",
          required: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS);
    },
    async (argv) => {
      checkBinary("sui", "sui");

      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const coreBridgeStateObjectId =
        argv["wormhole-state"] ?? CONTRACTS[network].sui.core;
      const tokenBridgeStateObjectId =
        argv["token-bridge-state"] ?? CONTRACTS[network].sui.token_bridge;
      const decimals = argv["decimals"];
      const provider = getProvider(
        network,
        argv.rpc ?? NETWORKS[network].sui.rpc
      );

      const res = await getCoinBuildOutputManual(
        provider,
        network,
        coreBridgeStateObjectId,
        tokenBridgeStateObjectId,
        decimals
      );
      console.log(res);
    }
  );
