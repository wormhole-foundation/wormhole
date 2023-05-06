import {
  assertChain,
  ChainName,
  CHAINS,
  coalesceChainName,
  isEVMChain,
  isTerraChain,
  toChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { execute_algorand } from "../algorand";
import { execute_aptos } from "../aptos";
import { NETWORK_OPTIONS } from "../consts";
import { execute_evm } from "../evm";
import { execute_injective } from "../injective";
import { execute_near } from "../near";
import { execute_sei } from "../sei";
import { execute_solana } from "../solana";
import { submit as submitSui } from "../sui";
import { execute_terra } from "../terra";
import { assertNetwork } from "../utils";
import { assertKnownPayload, impossible, parse } from "../vaa";
import { execute_xpla } from "../xpla";

export const command = "submit <vaa>";
export const desc = "Execute a VAA";
export const builder = (y: typeof yargs) =>
  y
    .positional("vaa", {
      describe: "vaa",
      type: "string",
      demandOption: true,
    })
    .option("chain", {
      alias: "c",
      describe: "chain name",
      choices: Object.keys(CHAINS) as (keyof typeof CHAINS)[],
      demandOption: false,
    } as const)
    .option("network", NETWORK_OPTIONS)
    .option("contract-address", {
      alias: "a",
      describe: "Contract to submit VAA to (override config)",
      type: "string",
      demandOption: false,
    })
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      demandOption: false,
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const vaa_hex = String(argv.vaa);
  const buf = Buffer.from(vaa_hex, "hex");
  const parsed_vaa = parse(buf);

  assertKnownPayload(parsed_vaa);
  console.log(parsed_vaa.payload);

  const network = argv.network.toUpperCase();
  assertNetwork(network);

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
  const cli_chain = argv.chain;

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
    await execute_evm(
      parsed_vaa.payload,
      buf,
      network,
      chain,
      argv["contract-address"],
      argv.rpc
    );
  } else if (isTerraChain(chain)) {
    await execute_terra(parsed_vaa.payload, buf, network, chain);
  } else if (chain === "solana" || chain === "pythnet") {
    await execute_solana(parsed_vaa, buf, network, chain);
  } else if (chain === "algorand") {
    await execute_algorand(
      parsed_vaa.payload,
      new Uint8Array(Buffer.from(vaa_hex, "hex")),
      network
    );
  } else if (chain === "near") {
    await execute_near(parsed_vaa.payload, vaa_hex, network);
  } else if (chain === "injective") {
    await execute_injective(parsed_vaa.payload, buf, network);
  } else if (chain === "xpla") {
    await execute_xpla(parsed_vaa.payload, buf, network);
  } else if (chain === "sei") {
    await execute_sei(parsed_vaa.payload, buf, network);
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    await submitSui(parsed_vaa.payload, buf, network, argv.rpc);
  } else if (chain === "aptos") {
    await execute_aptos(
      parsed_vaa.payload,
      buf,
      network,
      argv["contract-address"],
      argv.rpc
    );
  } else if (chain === "wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (chain === "btc") {
    throw Error("btc is not supported yet");
  } else if (chain === "sei") {
    throw Error("sei is not supported yet");
  } else {
    // If you get a type error here, hover over `chain`'s type and it tells you
    // which cases are not handled
    impossible(chain);
  }
};
