import fs from "fs";
import crypto from "crypto";
import chalk from "chalk";
import { inspect } from "util";

import {
  METADATA_LENGTH,
  getMetadataSectionLength,
  inferCompilerVersion,
} from "../helpers/solcMetadata";
import {
  init,
  Deployment,
  getOperationDescriptor,
  getProvider,
  loadWormholeRelayerImplementations,
  ChainInfo,
  getWormholeRelayer,
} from "../helpers/env";

const processName = "verifyWormholeRelayerDeployedByteCode";
init();
const operation = getOperationDescriptor();

const WORMHOLE_RELAYER_SOLIDITY_COMPILER_OUTPUT =
  "./build-forge/WormholeRelayer.sol/WormholeRelayer.json";
const WORMHOLE_RELAYER_BASE_SOLIDITY_COMPILER_OUTPUT =
  "./build-forge/WormholeRelayerBase.sol/WormholeRelayerBase.json";

async function run() {
  console.log("Start! " + processName);

  const expectedImplementationAddresses: Deployment[] =
    loadWormholeRelayerImplementations();

  // Taken from https://eips.ethereum.org/EIPS/eip-1967#logic-contract-address
  // Also found in the reference implementation `ERC1967Upgrade` contract from openzeppelin contracts.
  const implementationStorageSlot =
    "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

  for (const chain of operation.operatingChains) {
    const provider = getProvider(chain);
    const proxy = await getWormholeRelayer(chain, provider);

    const deployedImplementationAddress = expectedImplementationAddresses.find(
      (deploy) => {
        return deploy.chainId === chain.chainId;
      },
    );

    if (deployedImplementationAddress === undefined) {
      console.error(
        "Failed to find implementation address for chain " + chain.chainId,
      );
      continue;
    }

    const rawAddress = await provider.getStorageAt(
      proxy.address,
      implementationStorageSlot,
    );
    const actualAddress = decodeAddressFrom32ByteWord(rawAddress);

    if (
      actualAddress.toLowerCase() !==
      deployedImplementationAddress.address.toLowerCase()
    ) {
      console.error(
        `Implementation address in proxy does not match expected address.
Actual address: ${actualAddress.toLowerCase()}
Expected address: ${deployedImplementationAddress.address.toLowerCase()} ${chain.chainId}`,
      );
      continue;
    }

    console.log(
      `Verifying bytecode deployed at ${deployedImplementationAddress.address} on chain ${chain.chainId}...`,
    );

    let deployedBytecode: Buffer;
    try {
      deployedBytecode = Buffer.from(
        strip0x(await provider.getCode(deployedImplementationAddress.address)),
        "hex",
      );
    } catch (error) {
      console.error(
        `Failed to retrieve deployed contract from chain scanner. Error: ${stringifyError(error)}`,
      );
      continue;
    }

    let implementation: SolidityCompilerOutput;
    try {
      implementation = JSON.parse(
        fs.readFileSync(WORMHOLE_RELAYER_SOLIDITY_COMPILER_OUTPUT, "utf8"),
      );
    } catch (error) {
      console.error(
        `Failed to read WormholeRelayer contract data. Error: ${stringifyError(error)}`,
      );
      continue;
    }

    let baseImplementation: SolidityCompilerOutput;
    try {
      baseImplementation = JSON.parse(
        fs.readFileSync(WORMHOLE_RELAYER_BASE_SOLIDITY_COMPILER_OUTPUT, "utf8"),
      );
    } catch (error) {
      console.error(
        `Failed to read WormholeRelayerBase contract data. Error: ${stringifyError(error)}`,
      );
      continue;
    }

    let expectedBytecode: Buffer;
    try {
      expectedBytecode = buildByteCode(
        chain,
        implementation,
        baseImplementation,
      );
    } catch (error) {
      console.error(
        `Failed to build bytecode for chain ${chain.chainId}. Error: ${stringifyError(error)}`,
      );
      continue;
    }

    const deployedCompilerVersion = inferCompilerVersion(deployedBytecode);
    const expectedCompilerVersion = inferCompilerVersion(expectedBytecode);
    if (deployedCompilerVersion !== expectedCompilerVersion) {
      console.error(
        chalk.red(
          `Bytecode verification failed for chain ${chain.chainId}! Expected the compiler version ${expectedCompilerVersion} but got ${deployedCompilerVersion}`,
        ),
      );
      continue;
    }

    // We'll ignore the metadata section in this comparison since it's immaterial to the semantics of the contract.
    const deployedMetadataLength = getMetadataSectionLength(deployedBytecode);
    const expectedMetadataLength = getMetadataSectionLength(expectedBytecode);

    // We'll check that their sizes are the same because we don't expect variation in the metadata section format.
    if (deployedMetadataLength !== expectedMetadataLength) {
      console.error(
        chalk.red(
          `Bytecode verification failed for chain ${chain.chainId}! Expected the compiler version ${expectedCompilerVersion} but got ${deployedCompilerVersion}`,
        ),
      );
      continue;
    }

    const deployedBytecodeTrimmed = deployedBytecode.subarray(
      0,
      deployedMetadataLength + METADATA_LENGTH,
    );
    const expectedBytecodeTrimmed = expectedBytecode.subarray(
      0,
      expectedMetadataLength + METADATA_LENGTH,
    );

    const deployedBytecodeHash = sha256sum(deployedBytecodeTrimmed);
    const expectedBytecodeHash = sha256sum(expectedBytecodeTrimmed);

    if (deployedBytecodeHash !== expectedBytecodeHash) {
      console.error(
        chalk.red(
          `Bytecode verification failed for chain ${chain.chainId}! Expected hash ${expectedBytecodeHash} but got ${deployedBytecodeHash}`,
        ),
      );
      continue;
    }

    console.log(
      chalk.green(
        `Bytecode verified for chain ${chain.chainId} matches. Hash: ${deployedBytecodeHash}`,
      ),
    );
  }
}

interface ImmutableReference {
  start: number;
  length: number;
}

interface AstNode {
  id: number;
  mutability: string;
  name: string;
  nodes: AstNode[];
}

interface SolidityCompilerOutput {
  abi: string;
  deployedBytecode: {
    object: string;
    immutableReferences: Record<string, ImmutableReference[]>;
  };
  ast: {
    nodes: AstNode[];
  };
}

const knownRefs = {
  // the wormhole core contract address on the respective chain
  wormhole_: (chain: ChainInfo, valueLength: number): Buffer => {
    return toBufferOfLength(strip0x(chain.wormholeAddress), valueLength);
  },
  // the wormhole chain id of the respective chain
  chainId_: (chain: ChainInfo, valueLength: number): Buffer => {
    return uint16To32BytesBuffer(chain.chainId);
  },
};
/**
 * The WormholeRelayer contract has two immutable references:
 * 1. the wormhole core contract address on the respective chain
 * 2. the wormhole chain id of the respective chain
 *
 * The contract deployed bytecode will have these values replaced by the values for the respective chain.
 *
 * The definitions of the references are in the WormholeRelayerBase contract, that's why we need the compiler
 * output of both contracts to build the bytecode.
 * @param chain wormhole chain id of the chain to build the bytecode for
 * @param implementation the solidity compiler output of the WormholeRelayer contract
 * @param baseImplementation the solidity compiler output of the WormholeRelayerBase contract
 * @returns the bytecode for the WormholeRelayer contract with the immutable references replaced
 *          by the values for the respective chain
 */
function buildByteCode(
  chain: ChainInfo,
  implementation: SolidityCompilerOutput,
  baseImplementation: SolidityCompilerOutput,
): Buffer {
  let byteCodeWithImmutablesReplaced = Buffer.from(
    strip0x(implementation.deployedBytecode.object),
    "hex",
  );
  const immutableReferences =
    implementation.deployedBytecode.immutableReferences;

  for (const [key, references] of Object.entries(immutableReferences)) {
    const ref = getAstNode(parseInt(key), baseImplementation.ast.nodes);

    if (ref === null) {
      throw Error(`Failed to find AstNode for key ${parseInt(key)}`);
    }

    if (!Object.keys(knownRefs).includes(ref.name)) {
      throw Error(`Unknown reference name ${ref.name}`);
    }

    const refName = ref.name as keyof typeof knownRefs;

    if (ref.mutability !== "immutable") {
      throw Error(`Reference ${ref.name} is not immutable`);
    }

    const referenceValue = knownRefs[refName](chain, references[0].length);

    byteCodeWithImmutablesReplaced = replaceImmutableReferences(
      byteCodeWithImmutablesReplaced,
      references,
      referenceValue,
    );
  }

  return byteCodeWithImmutablesReplaced;
}

function getAstNode(id: number, nodes: AstNode[]): AstNode | null {
  let node: AstNode | null = null;

  for (const n of nodes) {
    if (n.id === id) {
      node = n;
      break;
    }

    if (n.nodes) {
      node = getAstNode(id, n.nodes);
      if (node) break;
    }
  }

  return node;
}

function replaceImmutableReferences(
  deployedBytecode: Buffer,
  immutableReferences: ImmutableReference[],
  referenceValue: Buffer, //32 bytes
): Buffer {
  const byteCodeWithImmutablesReplaced = Buffer.from(deployedBytecode);
  for (const ref of immutableReferences) {
    if (ref.length !== 32) {
      throw new Error("Only 32 byte words supported for immutable references.");
    }
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(
      referenceValue.readBigUInt64BE(0),
      ref.start,
    );
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(
      referenceValue.readBigUInt64BE(8),
      ref.start + 8,
    );
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(
      referenceValue.readBigUInt64BE(16),
      ref.start + 16,
    );
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(
      referenceValue.readBigUInt64BE(24),
      ref.start + 24,
    );
  }

  return byteCodeWithImmutablesReplaced;
}

function toBufferOfLength(raw: string, length: number) {
  const buffer = Buffer.from(raw, "hex");

  if (buffer.length > length) {
    throw new Error(`Buffer is longer than expected`);
  }

  const padding = Buffer.alloc(length - buffer.length);
  return Buffer.concat([padding, buffer]);
}

function decodeAddressFrom32ByteWord(word: string) {
  const buffer = Buffer.from(strip0x(word), "hex");

  if (buffer.length !== 32) {
    throw new Error(`Buffer is not word sized. Actual size: ${buffer.length}`);
  }

  const twelveZeroBytes = Buffer.alloc(12, 0);
  if (
    buffer.compare(
      twelveZeroBytes,
      0,
      twelveZeroBytes.length,
      0,
      twelveZeroBytes.length,
    ) !== 0
  ) {
    throw new Error(
      `Could not decode word ${word} as an address. First twelve bytes are not zeroed out.`,
    );
  }

  return `0x${buffer.subarray(twelveZeroBytes.length, buffer.length).toString("hex")}`;
}

function uint16To32BytesBuffer(num: number): Buffer {
  const buff = Buffer.alloc(32);
  buff.writeUInt16BE(num, 30);
  return buff;
}

function strip0x(str: string) {
  return str.startsWith("0x") ? str.substring(2) : str;
}

function sha256sum(buff: Buffer) {
  return crypto.createHash("sha256").update(buff).digest("hex");
}

function stringifyError(error: unknown): string {
  if (error instanceof Error) {
    return error.stack ?? error.message;
  }

  return inspect(error);
}

run().then(() => console.log("Done! " + processName));
