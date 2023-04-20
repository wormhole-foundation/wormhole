import yargs from "yargs";
import {
  CHAINS,
  assertChain,
  toChainName,
  ChainName,
  isEVMChain,
  isTerraChain,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import * as vaa from "../vaa";

exports.command = "submit <vaa>";
exports.desc = "Execute a VAA";
exports.builder = (y: typeof yargs) => {
  return y
    .positional("vaa", {
      describe: "vaa",
      type: "string",
      required: true,
    })
    .option("chain", {
      alias: "c",
      describe: "chain name",
      type: "string",
      choices: Object.keys(CHAINS),
      required: false,
    })
    .option("network", {
      alias: "n",
      describe: "network",
      type: "string",
      choices: ["mainnet", "testnet", "devnet"],
      required: true,
    })
    .option("contract-address", {
      alias: "a",
      describe: "Contract to submit VAA to (override config)",
      type: "string",
      required: false,
    })
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      required: false,
    });
};
exports.handler = async (argv) => {
  const vaa_hex = String(argv.vaa);
  const buf = Buffer.from(vaa_hex, "hex");
  const parsed_vaa = vaa.parse(buf);

  vaa.assertKnownPayload(parsed_vaa);

  console.log(parsed_vaa.payload);

  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }

  // We figure out the target chain to submit the VAA to.
  // The VAA might specify this itself (for example a contract upgrade VAA
  // or a token transfer VAA), in which case we just submit the VAA to
  // that target chain.
  //
  // If the VAA does not have a target (e.g. chain registration VAAs or
  // guardian set upgrade VAAs), we require the '--chain' argument to be
  // set on the command line.
  //
  // As a sanity check, in the event that the VAA does specify a target
  // and the '--chain' argument is also set, we issue an error if those
  // two don't agree instead of silently taking the VAA's target chain.

  // get VAA chain
  const vaa_chain_id =
    "chain" in parsed_vaa.payload ? parsed_vaa.payload.chain : 0;
  assertChain(vaa_chain_id);
  const vaa_chain = toChainName(vaa_chain_id);

  // get chain from command line arg
  const cli_chain = argv["chain"];

  let chain: ChainName;
  if (cli_chain !== undefined) {
    assertChain(cli_chain);
    if (vaa_chain !== "unset" && cli_chain !== vaa_chain) {
      throw Error(
        `Specified target chain (${cli_chain}) does not match VAA target chain (${vaa_chain})`
      );
    }
    chain = coalesceChainName(cli_chain);
  } else {
    chain = vaa_chain;
  }

  if (chain === "unset") {
    throw Error(
      "This VAA does not specify the target chain, please provide it by hand using the '--chain' flag."
    );
  } else if (isEVMChain(chain)) {
    const evm = require("../evm");
    await evm.execute_evm(
      parsed_vaa.payload,
      buf,
      network,
      chain,
      argv["contract-address"],
      argv["rpc"]
    );
  } else if (isTerraChain(chain)) {
    const terra = require("../terra");
    await terra.execute_terra(parsed_vaa.payload, buf, network, chain);
  } else if (chain === "solana" || chain === "pythnet") {
    const solana = require("../solana");
    await solana.execute_solana(parsed_vaa, buf, network, chain);
  } else if (chain === "algorand") {
    const algorand = require("../algorand");
    await algorand.execute_algorand(
      parsed_vaa.payload,
      new Uint8Array(Buffer.from(vaa_hex, "hex")),
      network
    );
  } else if (chain === "near") {
    const near = require("../near");
    await near.execute_near(parsed_vaa.payload, vaa_hex, network);
  } else if (chain === "injective") {
    const injective = require("../injective");
    await injective.execute_injective(parsed_vaa.payload, buf, network);
  } else if (chain === "xpla") {
    const xpla = require("../xpla");
    await xpla.execute_xpla(parsed_vaa.payload, buf, network);
  } else if (chain === "sei") {
    const sei = require("../sei");    
    await sei.execute_sei(parsed_vaa.payload, buf, network);
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    throw Error("SUI is not supported yet");
  } else if (chain === "aptos") {
    const aptos = require("../aptos");
    await aptos.execute_aptos(
      parsed_vaa.payload,
      buf,
      network,
      argv["contract-address"],
      argv["rpc"]
    );
  } else if (chain === "wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (chain === "btc") {
    throw Error("btc is not supported yet");
  } else {
    // If you get a type error here, hover over `chain`'s type and it tells you
    // which cases are not handled
    vaa.impossible(chain);
  }
};
