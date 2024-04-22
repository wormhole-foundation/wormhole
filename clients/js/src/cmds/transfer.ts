import {
  isCosmWasmChain,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { impossible } from "../vaa";
import { transferEVM } from "../evm";
import { CHAIN_NAME_CHOICES, NETWORK_OPTIONS, NETWORKS } from "../consts";
import { assertNetwork } from "../utils";
import { transferTerra } from "../terra";
import { transferInjective } from "../injective";
import { transferXpla } from "../xpla";
import { transferSolana } from "../solana";
import { transferAlgorand } from "../algorand";
import { transferNear } from "../near";
import { transferSui } from "../chains/sui/transfer";
import { transferAptos } from "../aptos";

export const command = "transfer";
export const desc = "Transfer a token";
export const builder = (y: typeof yargs) =>
  y
    .option("src-chain", {
      describe: "source chain",
      choices: CHAIN_NAME_CHOICES,
      demandOption: true,
    })
    .option("dst-chain", {
      describe: "destination chain",
      choices: CHAIN_NAME_CHOICES,
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
  const srcChain = argv["src-chain"];
  const dstChain = argv["dst-chain"];
  if (srcChain === "unset") {
    throw new Error("source chain is unset");
  }
  if (dstChain === "unset") {
    throw new Error("destination chain is unset");
  }
  // TODO: support transfers to sei
  if (dstChain === "sei") {
    throw new Error("transfer to sei currently unsupported");
  }
  if (srcChain === dstChain) {
    throw new Error("source and destination chains can't be the same");
  }
  const amount = argv.amount;
  if (BigInt(amount) <= 0) {
    throw new Error("amount must be greater than 0");
  }
  const tokenAddr = argv["token-addr"];
  if (tokenAddr === "native" && isCosmWasmChain(srcChain)) {
    throw new Error(`token-addr must be specified for ${srcChain}`);
  }
  const dstAddr = argv["dst-addr"];
  const network = argv.network.toUpperCase();
  assertNetwork(network);
  const rpc = argv.rpc ?? NETWORKS[network][srcChain].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${srcChain}`);
  }
  if (isEVMChain(srcChain)) {
    await transferEVM(
      srcChain,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (isTerraChain(srcChain)) {
    await transferTerra(
      srcChain,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (srcChain === "solana" || srcChain === "pythnet") {
    await transferSolana(
      srcChain,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (srcChain === "algorand") {
    await transferAlgorand(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "near") {
    await transferNear(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "injective") {
    await transferInjective(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "xpla") {
    await transferXpla(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "sei") {
    throw new Error("sei is not supported yet");
  } else if (srcChain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (srcChain === "sui") {
    await transferSui(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "aptos") {
    await transferAptos(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (srcChain === "btc") {
    throw Error("btc is not supported yet");
  } else if (srcChain === "cosmoshub") {
    throw Error("cosmoshub is not supported yet");
  } else if (srcChain === "evmos") {
    throw Error("evmos is not supported yet");
  } else if (srcChain === "kujira") {
    throw Error("kujira is not supported yet");
  } else if (srcChain === "neutron") {
    throw Error("neutron is not supported yet");
  } else if (srcChain === "celestia") {
    throw Error("celestia is not supported yet");
  } else if (srcChain === "stargaze") {
    throw Error("stargaze is not supported yet");
  } else if (srcChain === "seda") {
    throw Error("seda is not supported yet");
  } else if (srcChain === "dymension") {
    throw Error("dymension is not supported yet");
  } else if (srcChain === "provenance") {
    throw Error("provenance is not supported yet");
  } else if (srcChain === "rootstock") {
    throw Error("rootstock is not supported yet");
  } else {
    // If you get a type error here, hover over `chain`'s type and it tells you
    // which cases are not handled
    impossible(srcChain);
  }
};
