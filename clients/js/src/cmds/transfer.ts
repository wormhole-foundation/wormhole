import {
  isCosmWasmChain,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { impossible } from "../vaa";
import { transferEVM } from "../evm";
import { CHAIN_NAME_CHOICES, NETWORK_OPTIONS, NETWORKS } from "../consts";
import { transferTerra } from "../terra";
import { transferInjective } from "../injective";
import { transferXpla } from "../xpla";
import { transferSolana } from "../solana";
import { transferAlgorand } from "../algorand";
import { transferNear } from "../near";
import { transferSui } from "../chains/sui/transfer";
import { transferAptos } from "../aptos";
import {
  Chain,
  PlatformToChains,
  chain,
  chainToPlatform,
  toChain,
} from "@wormhole-foundation/sdk-base";
import { getNetwork } from "../utils";

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
  const srcChain: Chain = argv["src-chain"];
  const dstChain: Chain = argv["dst-chain"];
  // TODO: support transfers to sei
  if (dstChain === "Sei") {
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
  if (tokenAddr === "native" && chainToPlatform(srcChain) === "Cosmwasm") {
    throw new Error(`token-addr must be specified for ${srcChain}`);
  }
  const dstAddr = argv["dst-addr"];
  const network = getNetwork(argv.network);
  const rpc = argv.rpc ?? NETWORKS[network][toChain(srcChain)].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${srcChain}`);
  }
  // if (isEVMChain(srcChain)) {
  if (chainToPlatform(srcChain) === "Evm") {
    await transferEVM(
      srcChain as PlatformToChains<"Evm">,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (srcChain === "Terra" || srcChain === "Terra2") {
    await transferTerra(
      srcChain,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else if (srcChain === "Solana" || srcChain === "Pythnet") {
    await transferSolana(
      srcChain,
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
  } else if (srcChain === "Xpla") {
    await transferXpla(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Sei") {
    throw new Error("sei is not supported yet");
  } else if (srcChain === "Osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (srcChain === "Sui") {
    await transferSui(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Aptos") {
    await transferAptos(dstChain, dstAddr, tokenAddr, amount, network, rpc);
  } else if (srcChain === "Wormchain") {
    throw Error("Wormchain is not supported yet");
  } else if (srcChain === "Btc") {
    throw Error("btc is not supported yet");
  } else if (srcChain === "Cosmoshub") {
    throw Error("cosmoshub is not supported yet");
  } else if (srcChain === "Evmos") {
    throw Error("evmos is not supported yet");
  } else if (srcChain === "Kujira") {
    throw Error("kujira is not supported yet");
  } else if (srcChain === "Neutron") {
    throw Error("neutron is not supported yet");
  } else if (srcChain === "Celestia") {
    throw Error("celestia is not supported yet");
  } else {
    throw new Error(`${srcChain} is not supported yet`);
  }
};
