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
import { assertKnownPayload, parse, Payload, VAA } from "../vaa";
import {
  CLI_CHAINS,
  CliChain,
  assertLiveChain,
  chainToCliChain,
  cliChainIdToChain,
  cliChainToChainId,
  cliChainToPlatform,
  getChainRpc,
  getNetwork,
  getNftBridgeContract,
  getTokenBridgeContract,
} from "../utils";
import { Network, PlatformToChains } from "@wormhole-foundation/sdk";
import { TERRA2, execute_terra2 } from "../chains/terra2";
import { isDeprecatedChain } from "../chains/deprecated";

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
      describe:
        "chain name. To see a list of supported chains, run `worm chains`",
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
  let vaa_chain: CliChain | undefined;
  if (vaa_chain_id !== 0) {
    // cliChainIdToChain also covers Terra2 (id 18), which the SDK removed
    vaa_chain = cliChainIdToChain(vaa_chain_id);
  }

  // get chain from command line arg; chainToCliChain validates the name
  const cli_chain = argv.chain ? chainToCliChain(argv.chain) : undefined;

  let chain: CliChain;
  if (cli_chain !== undefined) {
    if (vaa_chain && cli_chain !== vaa_chain) {
      throw Error(
        `Specified target chain (${cli_chain}) does not match VAA target chain (${vaa_chain})`
      );
    }
    chain = cli_chain;
  } else {
    if (!vaa_chain) {
      throw Error(
        `VAA does not specify a target chain and one was not provided, please specify one with --chain or -c`
      );
    }
    chain = vaa_chain;
  }

  // the VAA (or --chain) may name a chain the SDK dropped — we can decode
  // those, but there is nothing live to submit to
  assertLiveChain(chain, "VAAs cannot be submitted to it");

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
  chain: CliChain,
  rpc: string | undefined,
  contractAddress: string | undefined
) {
  if (chain === TERRA2) {
    await execute_terra2(parsedVaa.payload, buf, network);
  } else if (cliChainToPlatform(chain) === "Evm") {
    await execute_evm(
      parsedVaa.payload,
      buf,
      network,
      chain as PlatformToChains<"Evm">,
      contractAddress,
      rpc
    );
  } else if (cliChainToPlatform(chain) === "Solana") {
    await execute_solana(
      parsedVaa,
      buf,
      network,
      chain as PlatformToChains<"Solana">
    );
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
  } else if (chain === "Sei") {
    await submitSei(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "Sui") {
    await submitSui(parsedVaa.payload, buf, network, rpc);
  } else if (chain === "Aptos") {
    await execute_aptos(parsedVaa.payload, buf, network, contractAddress, rpc);
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
  // the skip chain is compared by chain id: the VAA may reference a chain the
  // SDK no longer knows (e.g. Terra2)
  let skip_chain_id: number;
  if (parsedVaa.payload.type === "RegisterChain") {
    skip_chain_id = parsedVaa.payload.emitterChain;
  } else if (parsedVaa.payload.type === "AttestMeta") {
    skip_chain_id = parsedVaa.payload.tokenChain;
  } else {
    throw Error(
      `Invalid VAA payload type (${parsedVaa.payload.type}), only "RegisterChain" and "AttestMeta" are supported with --all-chains`
    );
  }

  for (const chain of CLI_CHAINS) {
    if (isDeprecatedChain(chain)) {
      console.log(
        `Skipping ${chain} because it was dropped from the SDK and has no live bridge`
      );
      continue;
    }
    const rpc = getChainRpc(network, chain);
    if (cliChainToChainId(chain) === skip_chain_id) {
      console.log(`Skipping ${chain} because it's the origin chain`);
      continue;
    }
    if (!rpc) {
      console.log(`Skipping ${chain} because the rpc is not defined`);
      continue;
    }
    const tokenBridge = getTokenBridgeContract(network, chain);
    const nftBridge = getNftBridgeContract(network, chain);
    if (
      (parsedVaa.payload.module === "TokenBridge" && !tokenBridge) ||
      (parsedVaa.payload.module === "NFTBridge" && !nftBridge)
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
