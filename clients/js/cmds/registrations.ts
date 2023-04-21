import yargs from "yargs";
import {
  CHAINS,
  assertChain,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

exports.command = "registrations <network> <chain> <module>";
exports.desc = "Print chain registrations";
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
      describe: "Module to query (TokenBridge or NFTBridge)",
      type: "string",
      choices: ["NFTBridge", "TokenBridge"],
      required: true,
    });
};
exports.handler = async (argv) => {
  assertChain(argv["chain"]);
  const chain = argv.chain;
  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }
  const module = argv.module;
  if (module !== "TokenBridge" && module !== "NFTBridge") {
    throw Error(`Module must be TokenBridge or NFTBridge`);
  }
  if (isEVMChain(chain)) {
    const evm = require("../evm");
    await evm.query_registrations_evm(network, chain, module);
  } else if (isTerraChain(chain)) {
    const terra = require("../terra");
    await terra.query_registrations_terra(network, chain, module);    
  } else if (chain === "injective") {
    const injective = require("../injective");    
    await injective.query_registrations_injective(network, module);
  } else if (chain === "xpla") {
    const xpla = require("../xpla");    
    await xpla.query_registrations_xpla(network, module);    
  } else if (chain === "sei") {
    const sei = require("../sei");    
    await sei.query_registrations_sei(network, module);
  } else {
    throw Error(`Command not supported for chain ${chain}`);
  }
};
