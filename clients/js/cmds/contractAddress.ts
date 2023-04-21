import yargs from "yargs";
import {
  ChainId,
  ChainName,
  CHAINS,
  assertChain,
  isCosmWasmChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { impossible } from "../vaa";
import { CONTRACTS } from "../consts";

exports.command = "contract <network> <chain> <module>";
exports.desc = "Print contract address";
exports.builder = (y: typeof yargs) => {
  return y
    .positional("network", {
      describe: "network",
      type: "string",
      choices: ["mainnet", "testnet", "devnet"],
    })
    .positional("chain", {
      describe: "Chain to query",
      type: "string",
      choices: Object.keys(CHAINS),
    })
    .positional("module", {
      describe: "Module to query",
      type: "string",
      choices: ["Core", "NFTBridge", "TokenBridge"],
    })
    .option("emitter", {
      alias: "e",
      describe: "Print in emitter address format",
      type: "boolean",
      default: false,
      required: false,
    });
};
exports.handler = async (argv) => {
  assertChain(argv["chain"]);
  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }
  let chain = argv["chain"];
  let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
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
    addr = await getEmitterAddress(chain, addr);
  }
  console.log(addr);
};

export async function getEmitterAddress(chain: ChainId | ChainName, addr: string) {
  const emitter = require("@certusone/wormhole-sdk/lib/cjs/bridge/getEmitterAddress");
  if (chain === "solana" || chain === "pythnet") {
    // TODO: Create an isSolanaChain()
    addr = emitter.getEmitterAddressSolana(addr);
  } else if (isCosmWasmChain(chain)) {
    addr = await emitter.getEmitterAddressTerra(addr);
  } else if (chain === "algorand") {
    addr = emitter.getEmitterAddressAlgorand(BigInt(addr));
  } else if (chain === "near") {
    addr = emitter.getEmitterAddressNear(addr);
  } else {
    addr = emitter.getEmitterAddressEth(addr);
  }

  return addr;
}
