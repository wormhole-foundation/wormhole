import {
  assertChain,
  ChainId,
  ChainName,
  CHAINS,
  coalesceChainName,
  Contracts,
  CONTRACTS,
  isEVMChain,
  isTerraChain,
  toChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { execute_algorand } from "../algorand";
import { execute_aptos } from "../aptos";
import { submit as submitSei } from "../chains/sei";
import { submit as submitSui } from "../chains/sui";
import { NETWORK_OPTIONS } from "../consts";
import { execute_evm } from "../evm";
import { execute_injective } from "../injective";
import { execute_near } from "../near";
import { execute_solana } from "../solana";
import { execute_terra } from "../terra";
import { assertNetwork } from "../utils";
import { assertKnownPayload, impossible, parse, Payload, VAA } from "../vaa";
import { execute_xpla } from "../xpla";
import { NETWORKS } from "../consts";
import { Network } from "../utils";

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
      choices: Object.keys(CHAINS) as ChainName[],
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
    })
    .option("all-chains", {
      alias: "ac",
      describe:
        "Submit the VAA to all chains except for the origin chain specified in the payload",
      type: "boolean",
      default: false,
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

  if (argv["all-chains"]) {
    if (argv.rpc) {
      throw Error(`--rpc may not be specified with --all-chains`);
    }

    if (argv["contract-address"]) {
      throw Error(`--contract_address may not be specified with --all-chains`);
    }

    await submitToAll(vaa_hex, parsed_vaa, buf, network);
    return;
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

  await executeSubmit(
    vaa_hex,
    parsed_vaa,
    buf,
    network,
    chain,
    argv.rpc,
    argv["contract-address"]
  );
};

async function executeSubmit(
  vaaHex: string,
  parsedVaa: VAA<Payload>,
  buf: Buffer,
  network: Network,
  chain: ChainName,
  rpc: string | undefined,
  contractAddress: string | undefined
) {
  if (chain === "unset") {
    throw Error(
      "This VAA does not specify the target chain, please provide it by hand using the '--chain' flag."
    );
  } else if (isEVMChain(chain)) {
    await execute_evm(
      parsedVaa.payload,
      buf,
      network,
      chain,
      contractAddress,
      rpc
    );
  } else if (isTerraChain(chain)) {
    await execute_terra(parsedVaa.payload, buf, network, chain);
  } else if (chain === "solana" || chain === "pythnet") {
    await execute_solana(parsedVaa, buf, network, chain);
  } else if (chain === "algorand") {
    await execute_algorand(
      parsedVaa.payload,
      new Uint8Array(Buffer.from(vaaHex, "hex")),
      network
    );
  } else if (chain === "near") {
    await execute_near(parsedVaa.payload, vaaHex, network);
  } else if (chain === "injective") {
    await execute_injective(parsedVaa.payload, buf, network);
  } else if (chain === "xpla") {
    await execute_xpla(parsedVaa.payload, buf, network);
  } else if (chain === "sei") {
    await submitSei(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    await submitSui(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "aptos") {
    await execute_aptos(parsedVaa.payload, buf, network, contractAddress, rpc);
  } else if (chain === "wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (chain === "btc") {
    throw Error("btc is not supported yet");
  } else if (chain === "cosmoshub") {
    throw Error("Cosmoshub is not supported yet");
  } else if (chain === "evmos") {
    throw Error("Evmos is not supported yet");
  } else if (chain === "kujira") {
    throw Error("kujira is not supported yet");
  } else if (chain === "neutron") {
    throw Error("neutron is not supported yet");
  } else if (chain === "celestia") {
    throw Error("celestia is not supported yet");
  } else if (chain === "stargaze") {
    throw Error("stargaze is not supported yet");
  } else if (chain === "seda") {
    throw Error("seda is not supported yet");
  } else if (chain === "dymension") {
    throw Error("dymension is not supported yet");
  } else if (chain === "provenance") {
    throw Error("provenance is not supported yet");
  } else if (chain === "rootstock") {
    throw Error("rootstock is not supported yet");
  } else {
    // If you get a type error here, hover over `chain`'s type and it tells you
    // which cases are not handled
    impossible(chain);
  }
}

async function submitToAll(
  vaaHex: string,
  parsedVaa: VAA<Payload>,
  buf: Buffer,
  network: Network
) {
  let skip_chain: ChainName = "unset";
  if (parsedVaa.payload.type === "RegisterChain") {
    skip_chain = toChainName(parsedVaa.payload.emitterChain as ChainId);
  } else if (parsedVaa.payload.type === "AttestMeta") {
    skip_chain = toChainName(parsedVaa.payload.tokenChain as ChainId);
  } else {
    throw Error(
      `Invalid VAA payload type (${parsedVaa.payload.type}), only "RegisterChain" and "AttestMeta" are supported with --all-chains`
    );
  }

  for (const chainStr in CHAINS) {
    let chain = chainStr as ChainName;
    if (chain === "unset") {
      continue;
    }
    const n = NETWORKS[network][chain];
    const contracts: Contracts = CONTRACTS[network][chain];
    if (chain == skip_chain) {
      console.log(`Skipping ${chain} because it's the origin chain`);
      continue;
    }
    if (!n || !n.rpc) {
      console.log(`Skipping ${chain} because the rpc is not defined`);
      continue;
    }
    if (!contracts) {
      console.log(
        `Skipping ${chain} because the contract entry is not defined`
      );
      return true;
    }
    if (
      (parsedVaa.payload.module === "TokenBridge" && !contracts.token_bridge) ||
      (parsedVaa.payload.module === "NFTBridge" && !contracts.nft_bridge)
    ) {
      console.log(`Skipping ${chain} because the contract is not defined`);
      continue;
    }

    console.log(`Submitting VAA to ${chain} ${network}`);
    try {
      await executeSubmit(
        vaaHex,
        parsedVaa,
        buf,
        network,
        chain,
        undefined,
        undefined
      );
    } catch (e) {
      console.error(`Failed to submit to ${chain}: `, e);
    }
  }
}
