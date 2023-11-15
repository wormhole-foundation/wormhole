import fs from "fs";
import crypto from "crypto";
import chalk from "chalk";

import {
  init,
  Deployment,
  getOperationDescriptor,
  getProvider,
  loadWormholeRelayerImplementations,
  ChainInfo
} from "../helpers/env";

const processName = "verifyWormholeRelayerDeployedByteCode";
init();
const operation = getOperationDescriptor();

const WORMHOLE_RELAYER_ABI_PATH = "./build-forge/WormholeRelayer.sol/WormholeRelayer.json";
const WORMHOLE_RELAYER_BASE_ABI_PATH = "./build-forge/WormholeRelayerBase.sol/WormholeRelayerBase.json";

async function run() {
  console.log("Start! " + processName);

  const implementationAddresses: Deployment[] = loadWormholeRelayerImplementations();

  for (const chain of operation.operatingChains) {
    const deployedImplementationAddress = implementationAddresses.find((deploy) => {
      return deploy.chainId === chain.chainId;
    });
    
    if (!deployedImplementationAddress) {
      console.error("Failed to find implementation address for chain " + chain.chainId);
      continue;
    }
    
    console.log(`Verifying bytecode deployed at ${deployedImplementationAddress} on chain ${chain.chainId}...`);

    const provider = getProvider(chain);

    let deployedBytecode: Buffer;
    try {
      deployedBytecode = Buffer.from(
        strip0x(await provider.getCode(deployedImplementationAddress.address)),
        "hex"
      );
    } catch (e) {
      console.error(`Failed to retrieve deployed contract from chain scanner. Error: ${e}`);
      continue;
    }

    let implementation: SolidityCompilerOutput;
    try {
      implementation = JSON.parse(fs.readFileSync(WORMHOLE_RELAYER_ABI_PATH, "utf8"));
    } catch (e) {
      console.error(`Failed to read WormholeRelayer contract data. Error: ${e}`);
      continue;
    }

    let baseImplementation: SolidityCompilerOutput;
    try {
      baseImplementation = JSON.parse(fs.readFileSync(WORMHOLE_RELAYER_BASE_ABI_PATH, "utf8"));
    } catch (e) {
      console.error(`Failed to read WormholeRelayerBase contract data. Error: ${e}`);
      continue;
    }

    let expectedByteCode: Buffer;
    try {
      expectedByteCode = buildByteCode(chain, implementation, baseImplementation);
    } catch (error) {
      console.error(`Failed to build bytecode for chain ${chain.chainId}. Error: ${error}`);
      continue;
    }

    const deployedBytecodeHash = sha256sum(deployedBytecode);
    const expectedByteCodeHash = sha256sum(expectedByteCode);

    if (deployedBytecodeHash !== expectedByteCodeHash) {
      console.error(
        chalk.red(
          `Bytecode verification failed for chain ${chain.chainId}! Expected hash ${expectedByteCodeHash} but got ${deployedBytecodeHash}`
        )
      );
      continue;
    }

    console.log(chalk.green(`Bytecode verified for chain ${chain.chainId} matches. Hash: ${deployedBytecodeHash}`));
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
  }
}

const knownRefs = {
  // the wormhole core contract address on the respective chain
  "wormhole_": (chain: ChainInfo, valueLength: number): Buffer => {
    return toBufferOfLength(strip0x(chain.wormholeAddress), valueLength);
  },
  // the wormhole chain id of the respective chain
  "chainId_": (chain: ChainInfo, valueLength: number): Buffer => {
    return uint16To32BytesBuffer(chain.chainId);
  },
}

function buildByteCode(
  chain: ChainInfo,
  implementation: SolidityCompilerOutput,
  baseImplementation: SolidityCompilerOutput,
): Buffer {

  let byteCodeWithImmutablesReplaced = Buffer.from(strip0x(implementation.deployedBytecode.object), "hex");
  const immutableReferences = implementation.deployedBytecode.immutableReferences;
  
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
      referenceValue
    );
  }

  return byteCodeWithImmutablesReplaced;
}

function getAstNode(id: number, nodes: AstNode[]): AstNode|null {
  let node: AstNode|null = null;

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
  referenceValue: Buffer //32 bytes
): Buffer {
  const byteCodeWithImmutablesReplaced = Buffer.from(deployedBytecode);
  for (const ref of immutableReferences) {
    if (ref.length !== 32) {
      throw new Error("Only 32 byte words supported for immutable references.");
    }
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(referenceValue.readBigUInt64BE(0), ref.start);
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(referenceValue.readBigUInt64BE(8), ref.start + 8);
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(referenceValue.readBigUInt64BE(16), ref.start + 16);
    byteCodeWithImmutablesReplaced.writeBigUInt64BE(referenceValue.readBigUInt64BE(24), ref.start + 24);
  }

  return byteCodeWithImmutablesReplaced;
}

function toBufferOfLength(raw: string, length: number) {
  const buffer = Buffer.from(raw, "hex");

  if (buffer.length > length) {
    throw new Error(`Buffer is longer than expected`);
  }

  const padding = Buffer.alloc(length - buffer.length);
  const a = Buffer.concat([padding, buffer]);
  
  return a;
}

function uint16To32BytesBuffer(num: number): Buffer {
  const buff = Buffer.alloc(32);
  buff.writeUInt16BE(num, 30);
  return buff;
};

function strip0x(str: string) {
  return str.startsWith("0x") ? str.substring(2) : str;
}

function sha256sum(buff: Buffer) {
  return crypto.createHash("sha256").update(buff).digest("hex");
}

run().then(() => console.log("Done! " + processName));
