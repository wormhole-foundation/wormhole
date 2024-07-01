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
import { assertKnownPayload, impossible, parse, Payload, VAA } from "../vaa";
import { execute_xpla } from "../xpla";
import { NETWORKS } from "../consts";
import { chainToChain, getNetwork } from "../utils";
import {
  Chain,
  Network,
  PlatformToChains,
  assertChain,
  assertChainId,
  chainIdToChain,
  chainToPlatform,
  chains,
  contracts,
  toChain,
} from "@wormhole-foundation/sdk";

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
      type: "string",
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

  const network = getNetwork(argv.network);

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

  // if vaa_chain_id is 0, it means the chain is not specified in the VAA.
  // We don't have a notion of an unsupported chain, so we don't want to just assert.
  let vaa_chain;
  if (vaa_chain_id !== 0) {
    assertChainId(vaa_chain_id);
    vaa_chain = chainIdToChain(vaa_chain_id);
  }

  // get chain from command line arg
  const cli_chain = argv.chain ? chainToChain(argv.chain) : argv.chain;

  let chain: Chain;
  if (cli_chain !== undefined) {
    assertChain(cli_chain);
    if (vaa_chain && cli_chain !== vaa_chain) {
      throw Error(
        `Specified target chain (${cli_chain}) does not match VAA target chain (${vaa_chain})`
      );
    }
    chain = toChain(cli_chain);
  } else {
    if (!vaa_chain) {
      throw Error(
        `VAA does not specify a target chain and one was not provided, please specify one with --chain or -c`
      );
    }
    assertChain(vaa_chain);
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
  chain: Chain,
  rpc: string | undefined,
  contractAddress: string | undefined
) {
  if (chainToPlatform(chain) === "Evm") {
    await execute_evm(
      parsedVaa.payload,
      buf,
      network,
      chain as PlatformToChains<"Evm">,
      contractAddress,
      rpc
    );
  } else if (chain === "Terra" || chain === "Terra2") {
    await execute_terra(parsedVaa.payload, buf, network, chain);
  } else if (chain === "Solana" || chain === "Pythnet") {
    await execute_solana(parsedVaa, buf, network, chain);
  } else if (chain === "Algorand") {
    await execute_algorand(
      parsedVaa.payload,
      new Uint8Array(Buffer.from(vaaHex, "hex")),
      network
    );
  } else if (chain === "Near") {
    await execute_near(parsedVaa.payload, vaaHex, network);
  } else if (chain === "Injective") {
    await execute_injective(parsedVaa.payload, buf, network);
  } else if (chain === "Xpla") {
    await execute_xpla(parsedVaa.payload, buf, network);
  } else if (chain === "Sei") {
    await submitSei(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "Osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "Sui") {
    await submitSui(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "Aptos") {
    await execute_aptos(parsedVaa.payload, buf, network, contractAddress, rpc);
  } else if (chain === "Wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (chain === "Btc") {
    throw Error("btc is not supported yet");
  } else if (chain === "Cosmoshub") {
    throw Error("Cosmoshub is not supported yet");
  } else if (chain === "Evmos") {
    throw Error("Evmos is not supported yet");
  } else if (chain === "Kujira") {
    throw Error("kujira is not supported yet");
  } else if (chain === "Neutron") {
    throw Error("neutron is not supported yet");
  } else if (chain === "Celestia") {
    throw Error("celestia is not supported yet");
  } else {
    throw new Error(`Unsupported chain: ${chain}`);
  }
}

async function submitToAll(
  vaaHex: string,
  parsedVaa: VAA<Payload>,
  buf: Buffer,
  network: Network
) {
  let skip_chain: Chain;
  if (parsedVaa.payload.type === "RegisterChain") {
    skip_chain = toChain(parsedVaa.payload.emitterChain);
  } else if (parsedVaa.payload.type === "AttestMeta") {
    skip_chain = toChain(parsedVaa.payload.tokenChain);
  } else {
    throw Error(
      `Invalid VAA payload type (${parsedVaa.payload.type}), only "RegisterChain" and "AttestMeta" are supported with --all-chains`
    );
  }

  for (const chain of chains) {
    const n = NETWORKS[network][chain];
    if (chain == skip_chain) {
      console.log(`Skipping ${chain} because it's the origin chain`);
      continue;
    }
    if (!n || !n.rpc) {
      console.log(`Skipping ${chain} because the rpc is not defined`);
      continue;
    }
    if (
      (parsedVaa.payload.module === "TokenBridge" &&
        !contracts.tokenBridge.get(network, chain)) ||
      (parsedVaa.payload.module === "NFTBridge" &&
        !contracts.nftBridge.get(network, chain))
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
