import yargs from "yargs";
import {
  CHAINS,
  assertChain,
  coalesceChainId,
  isCosmWasmChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { NETWORKS } from '../networks';
import { CONTRACTS } from '../consts';
import { impossible } from '../vaa';

const chain_args = {
  describe: "Chain to query",
  type: "string",
  choices: Object.keys(CHAINS),
} as const;

const network_args = {
  describe: "network",
  type: "string",
  choices: ["mainnet", "testnet", "devnet"],
} as const;

type ModuleArgChoice = "Core" | "NFTBridge" | "TokenBridge";

const module_args = {
  describe: "Module to query",
  type: "string",
  choices: ["Core", "NFTBridge", "TokenBridge"] as ModuleArgChoice[],
} as const;

exports.command = "info";
exports.desc = "Contract/chain/rpc information utilities";
exports.builder = (y: typeof yargs) => {
  return y
  .command("chain-id <chain>", "Print the wormhole chain ID integer associated with the specified chain name", (yargs) => {
    return yargs
      .positional("chain", chain_args)
  }, (argv) => {
    assertChain(argv["chain"]);
    console.log(coalesceChainId(argv["chain"]));
  })
  .command("rpc <network> <chain>", "Print RPC address", (yargs) => {
    return yargs
      .positional("network", network_args)
      .positional("chain", chain_args);
  }, (argv) => {
    assertChain(argv["chain"]);
    const network = argv.network.toUpperCase();
    if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
      throw Error(`Unknown network: ${network}`);
    }
    console.log(NETWORKS[network][argv["chain"]].rpc);
  })
  .command("contract <network> <chain> <module>", "Print contract address", (yargs) => {
    return yargs
      .positional("network", network_args)
      .positional("chain", chain_args)
      .positional("module", module_args)
      .option("emitter", {
        alias: "e",
        describe: "Print in emitter address format",
        type: "boolean",
        default: false,
        required: false,
      });
  }, async (argv) => {
    assertChain(argv["chain"]);
    const network = argv.network.toUpperCase();
    if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
      throw Error(`Unknown network: ${network}`);
    }
    let chain = argv["chain"];
    let module = argv["module"] as ModuleArgChoice;
    let addr = "";
    switch (module) {
      case "Core":
        addr = CONTRACTS[network][chain]["core"];
        break;
      case "NFTBridge":
        addr = CONTRACTS[network][chain]["nft_bridge"];
        break;
      case "TokenBridge":
        addr = CONTRACTS[network][chain]["token_bridge"];
        break;
      default:
        impossible(module);
    }
    if (argv["emitter"]) {
      const emitter = require("@certusone/wormhole-sdk/lib/cjs/bridge/getEmitterAddress");
      if (chain === "solana" || chain === "pythnet") {
        // TODO: Create an isSolanaChain()
        addr = await emitter.getEmitterAddressSolana(addr);
      } else if (isCosmWasmChain(chain)) {
        addr = await emitter.getEmitterAddressTerra(addr);
      } else if (chain === "algorand") {
        addr = emitter.getEmitterAddressAlgorand(BigInt(addr));
      } else if (chain === "near") {
        addr = emitter.getEmitterAddressNear(addr);
      } else {
        addr = emitter.getEmitterAddressEth(addr);
      }
    }
    console.log(addr);
  })
};
