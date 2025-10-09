import { fromBech32, toHex } from "@cosmjs/encoding";
import base58 from "bs58";
import { sha3_256 } from "js-sha3";
import yargs from "yargs";
import { GOVERNANCE_CHAIN, GOVERNANCE_EMITTER } from "../consts";
import { chainToChain, evm_address } from "../utils";
import {
  ContractUpgrade,
  Payload,
  PortalRegisterChain,
  RecoverChainId,
  serialiseVAA,
  sign,
  TokenBridgeAttestMeta,
  VAA,
  WormholeRelayerSetDefaultDeliveryProvider,
} from "../vaa";
import {
  Chain,
  chainToPlatform,
  Platform,
  platforms,
  toChainId,
} from "@wormhole-foundation/sdk-base";

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
              describe:
                "Chain to register. To see a list of supported chains, run `worm chains`",
              type: "string",
              demandOption: false,
            } as const)
            .option("chain-id", {
              alias: "i",
              describe:
                "Chain to register. To see a list of supported chains, run `worm chains`",
              type: "number",
              demandOption: false,
            } as const)
            .option("platform", {
              alias: "p",
              describe: "Platform to encode the address by",
              choices: platforms,
              demandOption: false,
            })
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
          if (!(argv.chain || (argv["chain-id"] && argv.platform))) {
            throw new Error("chain or chain-id and platform are required");
          }
          const module = argv["module"];
          const emitterChain = argv.chain
            ? toChainId(chainToChain(argv.chain))
            : argv["chain-id"];
          if (emitterChain === undefined) {
            throw new Error("emitterChain is undefined");
          }
          let emitterAddress = argv.platform
            ? parseAddressByPlatform(argv.platform, argv["contract-address"])
            : argv.chain
            ? parseAddress(chainToChain(argv.chain), argv["contract-address"])
            : undefined;
          if (emitterAddress === undefined) {
            throw new Error("emitterAddress is undefined");
          }
          const payload: PortalRegisterChain<typeof module> = {
            module,
            type: "RegisterChain",
            chain: 0,
            emitterChain,
            emitterAddress,
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
              describe:
                "Chain to upgrade. To see a list of supported chains, run `worm chains`",
              type: "string",
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
          const chain = chainToChain(argv.chain);
          const module = argv["module"];
          const payload: ContractUpgrade = {
            module,
            type: "ContractUpgrade",
            chain: toChainId(chain),
            address: parseCodeAddress(chain, argv["contract-address"]),
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
              describe:
                "Emitter chain of the VAA. To see a list of supported chains, run `worm chains`",
              type: "string",
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
              describe:
                "Token's chain. To see a list of supported chains, run `worm chains`",
              type: "string",
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
          const emitter_chain = chainToChain(argv["emitter-chain"]);
          const chain = chainToChain(argv.chain);
          const payload: TokenBridgeAttestMeta = {
            module: "TokenBridge",
            type: "AttestMeta",
            chain: 0,
            tokenAddress: parseAddress(chain, argv["token-address"]),
            tokenChain: toChainId(chain),
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
              describe:
                "Chain of Wormhole Relayer contract. To see a list of supported chains, run `worm chains`",
              type: "string",
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
          const chain = chainToChain(argv.chain);
          const payload: WormholeRelayerSetDefaultDeliveryProvider = {
            module: "WormholeRelayer",
            type: "SetDefaultDeliveryProvider",
            chain: toChainId(chain),
            relayProviderAddress: parseAddress(
              chain,
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

function parseAddressByPlatform(platform: Platform, address: string): string {
  if (platform === "Evm") {
    return "0x" + evm_address(address);
  } else if (platform === "Cosmwasm") {
    return "0x" + toHex(fromBech32(address).data).padStart(64, "0");
  } else if (platform === "Solana") {
    return "0x" + toHex(base58.decode(address)).padStart(64, "0");
  } else if (platform === "Algorand") {
    // TODO: is there a better native format for algorand?
    return "0x" + evm_address(address);
  } else if (platform === "Near") {
    return "0x" + evm_address(address);
  } else if (platform === "Sui") {
    return "0x" + evm_address(address);
  } else if (platform === "Aptos") {
    if (/^(0x)?[0-9a-fA-F]+$/.test(address)) {
      return "0x" + evm_address(address);
    }

    return sha3_256(Buffer.from(address)); // address is hash of fully qualified type
  } else if (platform === "Btc") {
    throw Error("btc is not supported yet");
  } else {
    throw Error(`Unsupported platform: ${platform}`);
  }
}

function parseAddress(chain: Chain, address: string): string {
  if (chainToPlatform(chain) === "Evm") {
    return "0x" + evm_address(address);
  } else if (chainToPlatform(chain) === "Cosmwasm") {
    return "0x" + toHex(fromBech32(address).data).padStart(64, "0");
  } else if (chain === "Solana" || chain === "Pythnet") {
    return "0x" + toHex(base58.decode(address)).padStart(64, "0");
  } else if (chain === "Algorand") {
    // TODO: is there a better native format for algorand?
    return "0x" + evm_address(address);
  } else if (chain === "Near") {
    return "0x" + evm_address(address);
  } else if (chain === "Sui") {
    return "0x" + evm_address(address);
  } else if (chain === "Aptos") {
    if (/^(0x)?[0-9a-fA-F]+$/.test(address)) {
      return "0x" + evm_address(address);
    }

    return sha3_256(Buffer.from(address)); // address is hash of fully qualified type
  } else if (chain === "Btc") {
    throw Error("btc is not supported yet");
  } else if (chain === "Cosmoshub") {
    throw Error("cosmoshub is not supported yet");
  } else if (chain === "Evmos") {
    throw Error("evmos is not supported yet");
  } else if (chain === "Kujira") {
    throw Error("kujira is not supported yet");
  } else {
    throw Error(`Unsupported chain: ${chain}`);
  }
}

function parseCodeAddress(chain: Chain, address: string): string {
  if (chainToPlatform(chain) === "Cosmwasm") {
    return "0x" + parseInt(address, 10).toString(16).padStart(64, "0");
  } else {
    return parseAddress(chain, address);
  }
}
