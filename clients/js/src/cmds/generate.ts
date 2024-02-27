import "@wormhole-foundation/connect-sdk-evm";
import "@wormhole-foundation/connect-sdk-solana";
import {
  assertChain,
  ChainName,
  CHAINS,
  isCosmWasmChain,
  isEVMChain,
  toChainId,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { fromBech32, toHex } from "@cosmjs/encoding";
import * as sdk from "@wormhole-foundation/connect-sdk";
import base58 from "bs58";
import { sha3_256 } from "js-sha3";
import yargs from "yargs";
import { GOVERNANCE_CHAIN, GOVERNANCE_EMITTER } from "../consts";
import { evm_address } from "../utils";
import {
  ContractUpgrade,
  impossible,
  Other,
  Payload,
  PortalRegisterChain,
  RecoverChainId,
  serialiseVAA,
  sign,
  TokenBridgeAttestMeta,
  VAA,
  vaaDigest,
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
  v.signatures = sign(signers, vaaDigest(v));
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
      // NTT Transfer
      .command(
        "ntt-transfer",
        "Generate an NTT transfer VAA",
        (yargs) =>
          yargs
            .option("source-chain", {
              alias: "sc",
              describe: "Chain to send from",
              choices: Object.values(sdk.chains) as sdk.Chain[],
              demandOption: true,
            } as const)
            .option("source-emitter", {
              alias: "e",
              describe: "Emitter address on source chain",
              type: "string",
              demandOption: true,
            } as const)
            .option("token-address", {
              alias: "t",
              describe: "token to transfer",
              type: "string",
              demandOption: true,
            })
            .option("amount", {
              alias: "a",
              describe: "Amount of token to send",
              type: "number",
              demandOption: true,
            } as const)
            .option("receiver", {
              alias: "r",
              describe: "Address of receiver on destination chain",
              type: "string",
              demandOption: true,
            } as const)
            .option("destination-chain", {
              alias: "dc",
              describe: "Chain to send to",
              choices: Object.values(sdk.chains) as sdk.Chain[],
              demandOption: true,
            } as const),
        (argv) => {
          const srcChain = argv["source-chain"];
          const dstChain = argv["destination-chain"];

          const token = sdk.toUniversal(srcChain, argv["token-address"]);
          const receiver = sdk.toUniversal(dstChain, argv["receiver"]);
          const emitter = sdk.toUniversal(srcChain, argv["source-emitter"]);

          if (sdk.isNative(token.address)) throw "what";

          let vaa = sdk.createVAA("NTT:Transfer", {
            signatures: [],
            timestamp: 1,
            nonce: 1,
            emitterChain: srcChain,
            emitterAddress: emitter,
            sequence: BigInt(Math.floor(Math.random() * 100000000)),
            consistencyLevel: 0,
            payload: {
              normalizedAmount: {
                // TODO: lookup decimals for token
                decimals: 18,
                amount: BigInt(argv["amount"]),
              },
              sourceToken: token,
              recipientAddress: receiver,
              recipientChain: dstChain,
            },
            guardianSet: 0,
          });

          const signers = argv["guardian-secret"].split(",");

          // @ts-ignore -- complains about being read-only
          vaa.signatures = sign(signers, sdk.encoding.hex.encode(vaa.hash)).map(
            (sig) => {
              return {
                signature: sdk.Signature.decode(
                  sdk.encoding.hex.decode(sig.signature)
                ),
                guardianIndex: sig.guardianSetIndex,
              };
            }
          );
          console.log(sdk.encoding.hex.encode(sdk.serialize(vaa)));
        }
      )
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
      // Attest token
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
      // SetDefaultDeliveryProvider
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
