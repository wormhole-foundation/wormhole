import yargs from "yargs";
import { transferEVM } from "../evm";
import { NETWORK_OPTIONS } from "../consts";
import { transferInjective } from "../injective";
import { transferSolana } from "../solana";
import { transferAlgorand } from "../algorand";
import { transferNear } from "../near";
import { transferSui } from "../chains/sui/transfer";
import { transferAptos } from "../aptos";
import { PlatformToChains } from "@wormhole-foundation/sdk-base";
import {
  CliChain,
  assertLiveChain,
  chainToCliChain,
  cliChainToPlatform,
  getChainRpc,
  getNetwork,
} from "../utils";
import { TERRA2, transferTerra2 } from "../chains/terra2";

export const command = "transfer";
export const desc = "Transfer a token";
export const builder = (y: typeof yargs) =>
  y
    .option("src-chain", {
      describe:
        "source chain. To see a list of supported chains, run `worm chains`",
      type: "string",
      demandOption: true,
    })
    .option("dst-chain", {
      describe:
        "destination chain. To see a list of supported chains, run `worm chains`",
      type: "string",
      demandOption: true,
    })
    .option("dst-addr", {
      describe: "destination address",
      type: "string",
      demandOption: true,
    })
    .option("token-addr", {
      describe: "token address",
      type: "string",
      default: "native",
      defaultDescription: "native token",
      demandOption: false,
    })
    .option("amount", {
      describe: "token amount",
      type: "string",
      demandOption: true,
    })
    .option("network", NETWORK_OPTIONS)
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      demandOption: false,
    });

export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const srcChain: CliChain = chainToCliChain(argv["src-chain"]);
  const dstChain: CliChain = chainToCliChain(argv["dst-chain"]);
  // TODO: support transfers to sei
  if (dstChain === "Sei") {
    throw new Error("transfer to sei currently unsupported");
  }
  assertLiveChain(srcChain, "transfers from it are not supported");
  assertLiveChain(dstChain, "transfers to it are not supported");
  if (srcChain === dstChain) {
    throw new Error("source and destination chains can't be the same");
  }
  const amount = argv.amount;
  if (BigInt(amount) <= 0) {
    throw new Error("amount must be greater than 0");
  }
  const tokenAddr = argv["token-addr"];
  if (tokenAddr === "native" && cliChainToPlatform(srcChain) === "Cosmwasm") {
    throw new Error(`token-addr must be specified for ${srcChain}`);
  }
  const dstAddr = argv["dst-addr"];
  const network = getNetwork(argv.network);
  const rpc = argv.rpc ?? getChainRpc(network, srcChain);
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${srcChain}`);
  }
  if (srcChain === TERRA2) {
    await transferTerra2(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (cliChainToPlatform(srcChain) === "Evm") {
    await transferEVM(
      srcChain as PlatformToChains<"Evm">,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (cliChainToPlatform(srcChain) === "Solana") {
    await transferSolana(
      srcChain as PlatformToChains<"Solana">,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (srcChain === "Algorand") {
    await transferAlgorand(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Near") {
    await transferNear(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Injective") {
    await transferInjective(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Sui") {
    await transferSui(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Aptos") {
    await transferAptos(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else {
    throw new Error(`${srcChain} is not supported yet`);
  }
};
