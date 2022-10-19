import { BCS, FaucetClient } from "aptos";
import { ethers } from "ethers";
import yargs from "yargs";
import { callEntryFunc, deriveResourceAccount, deriveWrappedAssetAddress } from "../aptos";
import { spawnSync } from 'child_process';
import { config } from '../config';
import fs from 'fs';
import sha3 from 'js-sha3';
import { NETWORKS } from "../networks";
import { assertChain, CHAIN_ID_APTOS, CHAIN_ID_SOLANA, coalesceChainId } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

type Network = "MAINNET" | "TESTNET" | "DEVNET"

interface Package {
  meta_file: string,
  mv_files: string[]
}

interface PackageBCS {
  meta: Uint8Array,
  bytecodes: Uint8Array,
  codeHash: Uint8Array
}

const network_options = {
  alias: "n",
  describe: "network",
  type: "string",
  choices: ["mainnet", "testnet", "devnet"],
  required: true,
} as const;

const rpc_description = {
  alias: "r",
  describe: "override default rpc endpoint url",
  type: "string",
  required: false,
} as const;

const named_addresses = {
  describe: "named addresses in the format addr1=0x0,addr2=0x1,...",
  type: "string",
  require: false
} as const;

// TODO(csongor): this could be useful elsewhere
function assertNetwork(n: string): asserts n is Network {
  if (
    n !== "MAINNET" &&
    n !== "TESTNET" &&
    n !== "DEVNET"
  ) {
    throw Error(`Unknown network: ${n}`);
  }
}

exports.command = 'aptos';
exports.desc = 'Aptos utilities ';
exports.builder = function(y: typeof yargs) {
  return y
    .command("init-token-bridge", "Init token bridge contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        // TODO(csongor): once the sdk has this, just use it from there
        .option("contract-address", {
          alias: "a",
          required: true,
          describe: "Address where the wormhole module is deployed",
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const contract_address = evm_address(argv["contract-address"]);
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      await callEntryFunc(network, rpc, `${contract_address}::token_bridge`, "init", [], []);
    })
    .command("init-wormhole", "Init Wormhole core contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("chain-id", {
          describe: "Chain id",
          type: "number",
          default: CHAIN_ID_APTOS,
          required: false
        })
        .option("governance-chain-id", {
          describe: "Governance chain id",
          type: "number",
          default: CHAIN_ID_SOLANA,
          required: false
        })
        .option("governance-address", {
          describe: "Governance address",
          type: "string",
          default: "0x0000000000000000000000000000000000000000000000000000000000000004",
          required: false
        })
        // TODO(csongor): once the sdk has this, just use it from there
        .option("contract-address", {
          alias: "a",
          required: true,
          describe: "Address where the wormhole module is deployed",
          type: "string",
        })
        .option("guardian-address", {
          alias: "g",
          required: true,
          describe: "Initial guardian's addresses (CSV)",
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);

      const contract_address = evm_address(argv["contract-address"]);
      const guardian_addresses = argv["guardian-address"].split(",").map(address => evm_address(address).substring(24));
      const chain_id = argv["chain-id"];
      const governance_address = evm_address(argv["governance-address"]);
      const governance_chain_id = argv["governance-chain-id"];

      const guardians_serializer = new BCS.Serializer();
      guardians_serializer.serializeU32AsUleb128(guardian_addresses.length);
      guardian_addresses.forEach(address => guardians_serializer.serializeBytes(Buffer.from(address, "hex")));

      const args = [
        BCS.bcsSerializeUint64(chain_id),
        BCS.bcsSerializeUint64(governance_chain_id),
        BCS.bcsSerializeBytes(Buffer.from(governance_address, "hex")),
        guardians_serializer.getBytes()
      ]
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      await callEntryFunc(network, rpc, `${contract_address}::wormhole`, "init", [], args);
    })
    .command("deploy <package-dir>", "Deploy an Aptos package", (yargs) => {
      return yargs
        .positional("package-dir", {
          type: "string"
        })
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("named-addresses", named_addresses)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      checkAptosBinary();
      const p = buildPackage(argv["package-dir"], argv["named-addresses"]);
      const b = serializePackage(p);
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      await callEntryFunc(network, rpc, "0x1::code", "publish_package_txn", [], [b.meta, b.bytecodes])
      console.log("Deployed:", p.mv_files)
    })
    .command("deploy-resource <seed> <package-dir>", "Deploy an Aptos package using a resource account", (yargs) => {
      return yargs
        .positional("seed", {
          type: "string"
        })
        .positional("package-dir", {
          type: "string"
        })
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("named-addresses", named_addresses)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      checkAptosBinary();
      const p = buildPackage(argv["package-dir"], argv["named-addresses"]);
      const b = serializePackage(p);
      const seed = Buffer.from(argv["seed"], "ascii")

      // TODO(csongor): use deployer address from sdk (when it's there)
      let module_name = "0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b::deployer";
      if (network == "TESTNET" || network == "MAINNET") {
        module_name = "0x0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b::deployer";
      }
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      await callEntryFunc(
        network,
        rpc,
        module_name,
        "deploy_derived",
        [],
        [
          b.meta,
          b.bytecodes,
          BCS.bcsSerializeBytes(seed)
        ])
      console.log("Deployed:", p.mv_files)
    })
    .command("send-example-message <message>", "Send example message", (yargs) => {
      return yargs
        .positional("message", {
          type: "string"
        })
        .option("network", network_options)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const rpc = NETWORKS[network]["aptos"].rpc;
      // TODO(csongor): use sdk address
      let module_name = "0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b::sender";
      if (network == "TESTNET" || network == "MAINNET") {
        module_name = "0x0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b::sender";
      }
      await callEntryFunc(network, rpc, module_name, "send_message", [], [BCS.bcsSerializeBytes(Buffer.from(argv["message"], "ascii"))])
    })
    .command("derive-resource-account <account> <seed>", "Derive resource account address", (yargs) => {
      return yargs
        .positional("account", {
          type: "string"
        })
        .positional("seed", {
          type: "string"
        })
    }, async (argv) => {
      console.log(deriveResourceAccount(Buffer.from(hex(argv['account']).substring(2), 'hex'), argv['seed']))
    })
    .command("derive-wrapped-address <chain> <origin-address>", "Derive wrapped coin type", (yargs) => {
      return yargs
        .positional("chain", {
          type: "string"
        })
        .positional("origin-address", {
          type: "string"
        })
        .option("network", network_options)
    }, async (argv) => {
      // TODO(csongor): this should be pulled in from the sdk.
      let token_bridge_address = Buffer.from("576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f", "hex");
      assertChain(argv["chain"]);
      let chain = coalesceChainId(argv["chain"]);
      let origin_address = Buffer.from(evm_address(argv["origin-address"]), "hex");
      console.log(deriveWrappedAssetAddress(token_bridge_address, chain, origin_address))
    })
    .command("hash-contracts <package-dir>", "Hash contract bytecodes for upgrade", (yargs) => {
      return yargs
        .positional("seed", {
          type: "string"
        })
        .positional("package-dir", {
          type: "string"
        })
        .option("named-addresses", named_addresses)
    }, (argv) => {
      checkAptosBinary();
      const p = buildPackage(argv["package-dir"], argv["named-addresses"]);
      const b = serializePackage(p);
      console.log(Buffer.from(b.codeHash).toString("hex"));
    })
    .command("upgrade <package-dir>", "Perform upgrade after VAA has been submitted", (_yargs) => {
      return yargs
        .positional("package-dir", {
          type: "string"
        })
        // TODO(csongor): once the sdk has the addresses, just look that up
        // based on the module
        .option("contract-address", {
          alias: "a",
          required: true,
          describe: "Address where the wormhole module is deployed",
          type: "string",
        })
        .option("network", network_options)
        .option("rpc", rpc_description)
        .option("named-addresses", named_addresses)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      checkAptosBinary();
      const p = buildPackage(argv["package-dir"], argv["named-addresses"]);
      const b = serializePackage(p);
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      // TODO(csongor): use deployer address from sdk (when it's there)
      const hash = await callEntryFunc(
        network,
        rpc,
        `${argv["contract-address"]}::contract_upgrade`,
        "upgrade",
        [],
        [
          b.meta,
          b.bytecodes,
        ])
      console.log("Deployed:", p.mv_files)
      console.log(hash)
    })
    .command("migrate", "Perform migration after contract upgrade", (_yargs) => {
      return yargs
        // TODO(csongor): once the sdk has the addresses, just look that up
        // based on the module
        .option("contract-address", {
          alias: "a",
          required: true,
          describe: "Address where the wormhole module is deployed",
          type: "string",
        })
        .option("network", network_options)
        .option("rpc", rpc_description)
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      checkAptosBinary();
      const rpc = argv.rpc ?? NETWORKS[network]["aptos"].rpc;
      // TODO(csongor): use deployer address from sdk (when it's there)
      const hash = await callEntryFunc(
        network,
        rpc,
        `${argv["contract-address"]}::contract_upgrade`,
        "migrate",
        [],
        [])
      console.log(hash)
    })
    // TODO - make faucet support testnet in additional to localnet
    .command("faucet", "Request money from the faucet for a given account", (yargs) => {
      return yargs
        .option("rpc", rpc_description)
        .option("faucet", {
          alias: "f",
          required: false,
          describe: "faucet url",
          type: "string",
        })
        .option("account", {
          alias: "a",
          required: false,
          describe: "account to fund",
          type: "string",
        })
    },
      async (argv) => {
        let NODE_URL = "http://0.0.0.0:8080/v1";
        let FAUCET_URL = "http://0.0.0.0:8081";
        let account = "0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b";

        if (argv.faucet != undefined) {
          FAUCET_URL = argv.faucet as string;
        }
        if (argv.rpc != undefined) {
          NODE_URL = argv.rpc as string;
        }
        if (argv.account != undefined) {
          account = argv.account as string;
        }
        const faucetClient = new FaucetClient(NODE_URL, FAUCET_URL);
        const coins = 20000000;
        await faucetClient.fundAccount(account, coins);
        console.log(`Funded ${account} with ${coins} coins`);
      })
    .strict().demandCommand();
}

function hex(x: string): string {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true });
}
function evm_address(x: string): string {
  return hex(x).substring(2).padStart(64, "0");
}

export function checkAptosBinary(): void {
  const dir = `${config.wormholeDir}/aptos`;
  const aptos = spawnSync("aptos", ["--version"]);
  if (aptos.status !== 0) {
    console.error("aptos is not installed. Please install aptos and try again.");
    console.error(`See ${dir}/README.md for instructions.`);
    process.exit(1);
  }
}

function buildPackage(dir: string, addrs?: string): Package {
  const named_addresses =
    addrs
      ? ["--named-addresses", addrs]
      : [];
  const aptos = spawnSync("aptos",
    ["move", "compile", "--save-metadata", "--included-artifacts", "none", "--package-dir", dir, ...named_addresses])
  if (aptos.status !== 0) {
    console.error(aptos.stderr.toString('utf8'))
    console.error(aptos.stdout.toString('utf8'))
    process.exit(1)
  }

  const result: any = JSON.parse(aptos.stdout.toString('utf8'))
  const buildDirs =
    fs.readdirSync(`${dir}/build`, { withFileTypes: true })
      .filter(dirent => dirent.isDirectory())
      .map(dirent => dirent.name)
  if (buildDirs.length !== 1) {
    console.error(`Unexpected directory structure in ${dir}/build: expected a single directory`)
    process.exit(1)
  }
  const buildDir = `${dir}/build/${buildDirs[0]}`
  return {
    meta_file: `${buildDir}/package-metadata.bcs`,
    mv_files: result["Result"].map((mod: string) => `${buildDir}/bytecode_modules/${mod.split("::")[1]}.mv`)
  }
}

function serializePackage(p: Package): PackageBCS {
  const metaBytes = fs.readFileSync(p.meta_file);
  const packageMetadataSerializer = new BCS.Serializer();
  packageMetadataSerializer.serializeBytes(metaBytes)
  const serializedPackageMetadata = packageMetadataSerializer.getBytes();

  const modules = p.mv_files.map(file => fs.readFileSync(file))
  const serializer = new BCS.Serializer();
  serializer.serializeU32AsUleb128(modules.length);
  modules.forEach(module => serializer.serializeBytes(module));
  const serializedModules = serializer.getBytes();

  const hashes = [metaBytes].concat(modules).map((x) => Buffer.from(sha3.keccak256(x), "hex"));
  const codeHash = Buffer.from(sha3.keccak256(Buffer.concat(hashes)), "hex")

  return {
    meta: serializedPackageMetadata,
    bytecodes: serializedModules,
    codeHash
  }
}
