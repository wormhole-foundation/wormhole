import { tryNativeToHexString } from "@certusone/wormhole-sdk/lib/esm/utils/array";
import {
  assertChain,
  ChainName,
  CHAINS,
  isCosmWasmChain,
  isEVMChain,
  toChainId,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { fromBech32, toHex } from "@cosmjs/encoding";
import base58 from "bs58";
import { sha3_256 } from "js-sha3";
import yargs from "yargs";
import { GOVERNANCE_CHAIN, GOVERNANCE_EMITTER } from "../consts";
import { evm_address } from "../utils";
import {
  ContractUpgrade,
  impossible,
  Payload,
  PortalRegisterChain,
  RecoverChainId,
  serialiseVAA,
  sign,
  TokenBridgeAttestMeta,
  VAA,
  WormholeRelayerSetDefaultDeliveryProvider,
} from "../vaa";

function makeVAA(
  emitterChain: number,
  emitterAddress: string,
  signers: string[],
  p: Payload
): VAA<Payload> {
  let v: VAA<Payload> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 1,
    nonce: 1,
    emitterChain: emitterChain,
    emitterAddress: emitterAddress,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: p,
  };
  v.signatures = sign(signers, v);
  return v;
}

export const command = "generate";
export const desc = "generate VAAs (devnet and testnet only)";
export const builder = function (y: typeof yargs) {
  return (
    y
      .option("guardian-secret", {
        alias: "g",
        demandOption: true,
        describe: "Guardians' secret keys (CSV)",
        type: "string",
      })
      // Registration
      .command(
        "registration",
        "Generate registration VAA",
        (yargs) =>
          yargs
            .option("chain", {
              alias: "c",
              describe: "Chain to register",
              choices: Object.keys(CHAINS) as ChainName[],
              demandOption: true,
            } as const)
            .option("contract-address", {
              alias: "a",
              describe: "Contract to register",
              type: "string",
              demandOption: true,
            })
            .option("module", {
              alias: "m",
              describe: "Module to register",
              choices: ["NFTBridge", "TokenBridge", "WormholeRelayer"],
              demandOption: true,
            } as const),
        (argv) => {
          const module = argv["module"];
          assertChain(argv.chain);
          const payload: PortalRegisterChain<typeof module> = {
            module,
            type: "RegisterChain",
            chain: 0,
            emitterChain: toChainId(argv.chain),
            emitterAddress: parseAddress(argv.chain, argv["contract-address"]),
          };
          const vaa = makeVAA(
            GOVERNANCE_CHAIN,
            GOVERNANCE_EMITTER,
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(vaa));
        }
      )
      // Upgrade
      .command(
        "upgrade",
        "Generate contract upgrade VAA",
        (yargs) =>
          yargs
            .option("chain", {
              alias: "c",
              describe: "Chain to upgrade",
              choices: Object.keys(CHAINS) as ChainName[],
              demandOption: true,
            } as const)
            .option("contract-address", {
              alias: "a",
              describe: "Contract to upgrade to",
              type: "string",
              demandOption: true,
            })
            .option("module", {
              alias: "m",
              describe: "Module to upgrade",
              choices: ["Core", "NFTBridge", "TokenBridge", "WormholeRelayer"],
              demandOption: true,
            } as const),
        (argv) => {
          assertChain(argv.chain);
          const module = argv["module"];
          const payload: ContractUpgrade = {
            module,
            type: "ContractUpgrade",
            chain: toChainId(argv.chain),
            address: parseCodeAddress(argv.chain, argv["contract-address"]),
          };
          const vaa = makeVAA(
            GOVERNANCE_CHAIN,
            GOVERNANCE_EMITTER,
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(vaa));
        }
      )
      .command(
        "attestation",
        "Generate a token attestation VAA",
        (yargs) =>
          yargs
            .option("emitter-chain", {
              alias: "e",
              describe: "Emitter chain of the VAA",
              choices: Object.keys(CHAINS) as ChainName[],
              demandOption: true,
            } as const)
            .option("emitter-address", {
              alias: "f",
              describe: "Emitter address of the VAA",
              type: "string",
              demandOption: true,
            })
            .option("chain", {
              alias: "c",
              describe: "Token's chain",
              choices: Object.keys(CHAINS) as ChainName[],
              demandOption: true,
            } as const)
            .option("token-address", {
              alias: "a",
              describe: "Token's address",
              type: "string",
              demandOption: true,
            })
            .option("decimals", {
              alias: "d",
              describe: "Token's decimals",
              type: "number",
              demandOption: true,
            })
            .option("symbol", {
              alias: "s",
              describe: "Token's symbol",
              type: "string",
              demandOption: true,
            })
            .option("name", {
              alias: "n",
              describe: "Token's name",
              type: "string",
              demandOption: true,
            }),
        (argv) => {
          const emitter_chain = argv["emitter-chain"];
          assertChain(argv.chain);
          assertChain(emitter_chain);
          const payload: TokenBridgeAttestMeta = {
            module: "TokenBridge",
            type: "AttestMeta",
            chain: 0,
            tokenAddress: parseAddress(argv.chain, argv["token-address"]),
            tokenChain: toChainId(argv.chain),
            decimals: argv["decimals"],
            symbol: argv["symbol"],
            name: argv["name"],
          };
          const vaa = makeVAA(
            toChainId(emitter_chain),
            parseAddress(emitter_chain, argv["emitter-address"]),
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(vaa));
        }
      )
      // RecoverChainId
      .command(
        "recover-chain-id",
        "Generate a recover chain ID VAA",
        (yargs) =>
          yargs
            .option("module", {
              alias: "m",
              describe: "Module to recover",
              choices: ["Core", "NFTBridge", "TokenBridge"],
              demandOption: true,
            } as const)
            .option("evm-chain-id", {
              alias: "e",
              describe: "EVM chain ID to set",
              type: "string",
              demandOption: true,
            })
            .option("new-chain-id", {
              alias: "c",
              describe: "New chain ID to set",
              type: "number",
              demandOption: true,
            }),
        (argv) => {
          const module = argv["module"];
          const payload: RecoverChainId = {
            module,
            type: "RecoverChainId",
            evmChainId: BigInt(argv["evm-chain-id"]),
            newChainId: argv["new-chain-id"],
          };
          const vaa = makeVAA(
            GOVERNANCE_CHAIN,
            GOVERNANCE_EMITTER,
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(vaa));
        }
      )
      .command(
        "set-default-delivery-provider",
        "Sets the default delivery provider for the Wormhole Relayer contract",
        (yargs) => {
          return yargs
            .option("chain", {
              alias: "c",
              describe: "Chain of Wormhole Relayer contract",
              choices: Object.keys(CHAINS),
              demandOption: true,
            } as const)
            .option("delivery-provider-address", {
              alias: "p",
              describe: "Address of the delivery provider contract",
              type: "string",
              demandOption: true,
            });
        },
        (argv) => {
          assertChain(argv.chain);
          const payload: WormholeRelayerSetDefaultDeliveryProvider = {
            module: "WormholeRelayer",
            type: "SetDefaultDeliveryProvider",
            chain: toChainId(argv["chain"]),
            relayProviderAddress: parseAddress(
              argv["chain"],
              argv["delivery-provider-address"]
            ),
          };
          let v = makeVAA(
            GOVERNANCE_CHAIN,
            GOVERNANCE_EMITTER,
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(v));
        }
      )
  );
};
export const handler = () => {};

function parseAddress(chain: ChainName, address: string): string {
  if (chain === "unset") {
    throw Error("Chain unset");
  } else if (isEVMChain(chain)) {
    return "0x" + evm_address(address);
  } else if (isCosmWasmChain(chain)) {
    return "0x" + toHex(fromBech32(address).data).padStart(64, "0");
  } else if (chain === "solana" || chain === "pythnet") {
    return "0x" + toHex(base58.decode(address)).padStart(64, "0");
  } else if (chain === "algorand") {
    // TODO: is there a better native format for algorand?
    return "0x" + evm_address(address);
  } else if (chain === "near") {
    return "0x" + evm_address(address);
  } else if (chain === "sui") {
    return "0x" + evm_address(address);
  } else if (chain === "aptos") {
    if (/^(0x)?[0-9a-fA-F]+$/.test(address)) {
      return "0x" + evm_address(address);
    }

    return sha3_256(Buffer.from(address)); // address is hash of fully qualified type
  } else if (chain === "btc") {
    throw Error("btc is not supported yet");
  } else if (chain === "cosmoshub") {
    throw Error("cosmoshub is not supported yet");
  } else if (chain === "evmos") {
    throw Error("evmos is not supported yet");
  } else if (chain === "kujira") {
    throw Error("kujira is not supported yet");
  } else if (chain === "rootstock") {
    throw Error("rootstock is not supported yet");
  } else {
    impossible(chain);
  }
}

function parseCodeAddress(chain: ChainName, address: string): string {
  if (isCosmWasmChain(chain)) {
    return "0x" + parseInt(address, 10).toString(16).padStart(64, "0");
  } else {
    return parseAddress(chain, address);
  }
}
