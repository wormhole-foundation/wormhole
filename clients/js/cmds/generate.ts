import {
  assertChain,
  ChainName,
  CHAINS,
  isCosmWasmChain,
  isEVMChain,
  toChainId,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { fromBech32, toHex } from "@cosmjs/encoding";
import base58 from "bs58";
import { sha3_256 } from "js-sha3";
import yargs from "yargs";
import { evm_address, hex } from "../consts";
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

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";

exports.command = "generate";
exports.desc = "generate VAAs (devnet and testnet only)";
exports.builder = function (y: typeof yargs) {
  return (
    y
      .option("guardian-secret", {
        alias: "g",
        required: true,
        describe: "Guardians' secret keys (CSV)",
        type: "string",
      })
      // Registration
      .command(
        "registration",
        "Generate registration VAA",
        (yargs) => {
          return yargs
            .option("chain", {
              alias: "c",
              describe: "Chain to register",
              type: "string",
              choices: Object.keys(CHAINS),
              required: true,
            })
            .option("contract-address", {
              alias: "a",
              describe: "Contract to register",
              type: "string",
              required: true,
            })
            .option("module", {
              alias: "m",
              describe: "Module to upgrade",
              type: "string",
              choices: ["NFTBridge", "TokenBridge"],
              required: true,
            });
        },
        (argv) => {
          let module = argv["module"] as "NFTBridge" | "TokenBridge";
          assertChain(argv["chain"]);
          let payload: PortalRegisterChain<typeof module> = {
            module,
            type: "RegisterChain",
            chain: 0,
            emitterChain: toChainId(argv["chain"]),
            emitterAddress: parseAddress(
              argv["chain"],
              argv["contract-address"]
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
      // Upgrade
      .command(
        "upgrade",
        "Generate contract upgrade VAA",
        (yargs) => {
          return yargs
            .option("chain", {
              alias: "c",
              describe: "Chain to upgrade",
              type: "string",
              choices: Object.keys(CHAINS),
              required: true,
            })
            .option("contract-address", {
              alias: "a",
              describe: "Contract to upgrade to",
              type: "string",
              required: true,
            })
            .option("module", {
              alias: "m",
              describe: "Module to upgrade",
              type: "string",
              choices: ["Core", "NFTBridge", "TokenBridge"],
              required: true,
            });
        },
        (argv) => {
          assertChain(argv["chain"]);
          let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
          let payload: ContractUpgrade = {
            module,
            type: "ContractUpgrade",
            chain: toChainId(argv["chain"]),
            address: parseCodeAddress(argv["chain"], argv["contract-address"]),
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
      .command(
        "attestation",
        "Generate a token attestation VAA",
        (yargs) => {
          return yargs
            .option("emitter-chain", {
              alias: "e",
              describe: "Emitter chain of the VAA",
              type: "string",
              choices: Object.keys(CHAINS),
              required: true,
            })
            .option("emitter-address", {
              alias: "f",
              describe: "Emitter address of the VAA",
              type: "string",
              required: true,
            })
            .option("chain", {
              alias: "c",
              describe: "Token's chain",
              type: "string",
              choices: Object.keys(CHAINS),
              required: true,
            })
            .option("token-address", {
              alias: "a",
              describe: "Token's address",
              type: "string",
              required: true,
            })
            .option("decimals", {
              alias: "d",
              describe: "Token's decimals",
              type: "number",
              required: true,
            })
            .option("symbol", {
              alias: "s",
              describe: "Token's symbol",
              type: "string",
              required: true,
            })
            .option("name", {
              alias: "n",
              describe: "Token's name",
              type: "string",
              required: true,
            });
        },
        (argv) => {
          let emitter_chain = argv["emitter-chain"] as string;
          assertChain(argv["chain"]);
          assertChain(emitter_chain);
          let payload: TokenBridgeAttestMeta = {
            module: "TokenBridge",
            type: "AttestMeta",
            chain: 0,
            tokenAddress: parseAddress(argv["chain"], argv["token-address"]),
            tokenChain: toChainId(argv["chain"]),
            decimals: argv["decimals"],
            symbol: argv["symbol"],
            name: argv["name"],
          };
          let v = makeVAA(
            toChainId(emitter_chain),
            parseAddress(emitter_chain, argv["emitter-address"] as string),
            argv["guardian-secret"].split(","),
            payload
          );
          console.log(serialiseVAA(v));
        }
      )
      // RecoverChainId
      .command(
        "recover-chain-id",
        "Generate a recover chain ID VAA",
        (yargs) => {
          return yargs
            .option("module", {
              alias: "m",
              describe: "Module to upgrade",
              type: "string",
              choices: ["Core", "NFTBridge", "TokenBridge"],
              required: true,
            })
            .option("evm-chain-id", {
              alias: "e",
              describe: "EVM chain ID to set",
              type: "string",
              required: true,
            })
            .option("new-chain-id", {
              alias: "c",
              describe: "New chain ID to set",
              type: "number",
              required: true,
            });
        },
        (argv) => {
          let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
          let payload: RecoverChainId = {
            module,
            type: "RecoverChainId",
            evmChainId: BigInt(argv["evm-chain-id"]),
            newChainId: argv["new-chain-id"],
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
    return "0x" + hex(address).substring(2).padStart(64, "0");
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    throw Error("SUI is not supported yet");
  } else if (chain === "aptos") {
    if (/^(0x)?[0-9a-fA-F]+$/.test(address)) {
      return "0x" + evm_address(address);
    }

    return sha3_256(Buffer.from(address)); // address is hash of fully qualified type
  } else if (chain === "wormchain") {
    const sdk = require("@certusone/wormhole-sdk/lib/cjs/utils/array");
    return "0x" + sdk.tryNativeToHexString(address, chain);
  } else if (chain === "btc") {
    throw Error("btc is not supported yet");
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
