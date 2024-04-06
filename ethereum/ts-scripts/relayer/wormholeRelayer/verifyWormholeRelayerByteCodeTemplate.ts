import fs from "fs";
import crypto from "crypto";
import chalk from "chalk";

import {
  init,
  Deployment,
  getOperationDescriptor,
  getProvider,
  loadWormholeRelayerImplementations,
} from "../helpers/env";

const processName = "verifyWormholeRelayerByteCodeTemplate";
init();
const operation = getOperationDescriptor();

const WORMHOLE_RELAYER_SOLIDITY_COMPILER_OUTPUT = "./build-forge/WormholeRelayer.sol/WormholeRelayer.json";
/**
 * This scripts serves to verify that the locally built bytecode for the WormholeRelayer contract
 * matches the bytecode deployed on already supported chains.
 * It's meant to be used as a helper script before deploying the WormholeRelayer contract to a new chain.
 * It should help verify that the implementation deployed to a new chain matches the implementation
 * previously deployed on another chains.
 * The WormholeRelayer contract uses immutable references that are replaced by different values on each chain.
 * In this script we'll replace all immutable references by 0 both on the SolidityCompilerOutput and on the 
 * built bytecode to compare the rest of the bytecode.
 * The script will compare the locally built bytcode at $WORMHOLE_RELAYER_SOLIDITY_COMPILER_OUTPUT with
 * the bytecode it pulls from each of the operatingChains.
 */
async function run() {
  console.log("Start! " + processName);

  let implementation: SolidityCompilerOutput;
  try {
    implementation = JSON.parse(fs.readFileSync(WORMHOLE_RELAYER_SOLIDITY_COMPILER_OUTPUT, "utf8"));
  } catch (e) {
    console.error(`Failed to read WormholeRelayer contract data. Error: ${e}`);
    throw e;
  }

  const immutableReferences = implementation.deployedBytecode.immutableReferences;

  let bytecode = Buffer.from(strip0x(implementation.deployedBytecode.object), "hex");
  const bytecodeTemplate = replaceAllImmutableReferencesBy0(bytecode, immutableReferences);

  const implementationAddresses: Deployment[] = loadWormholeRelayerImplementations();

  for (const chain of operation.operatingChains) {
    const deployedImplementationAddress = implementationAddresses.find((deploy) => {
      return deploy.chainId === chain.chainId;
    });
    
    if (!deployedImplementationAddress) {
      console.error("Failed to find implementation address for chain " + chain.chainId);
      continue;
    }
    
    console.log(`Verifying bytecode at ${deployedImplementationAddress.address} matches the local template...`);

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

    const deployedBytecodeTemplate = replaceAllImmutableReferencesBy0(deployedBytecode, immutableReferences);

    const deployedBytecodeHash = sha256sum(bytecodeTemplate);
    const expectedByteCodeHash = sha256sum(deployedBytecodeTemplate);

    if (deployedBytecodeHash !== expectedByteCodeHash) {
      console.error(
        chalk.red(
          `Bytecode verification failed for chain ${chain.chainId}! Expected hash ${expectedByteCodeHash} but got ${deployedBytecodeHash}`
        )
      );
      continue;
    }

    console.log(chalk.green(`Local bytecode matches the template deployed for chain ${chain.chainId}. Hash: ${deployedBytecodeHash}`));
  }
}

function replaceAllImmutableReferencesBy0 (bytecode: Buffer, referencesByKey: Record<string, ImmutableReference[]>) {
  let byteCodeWithImmutablesReplaced = Buffer.from(bytecode);
  for (const references of Object.values(referencesByKey)) {
    byteCodeWithImmutablesReplaced = replaceImmutableReferences(byteCodeWithImmutablesReplaced, references, Buffer.alloc(32));
  }
  return byteCodeWithImmutablesReplaced;
}

interface ImmutableReference {
  start: number;
  length: number;
}

interface SolidityCompilerOutput {
  abi: string;
  deployedBytecode: {
    object: string;
    immutableReferences: Record<string, ImmutableReference[]>;
  };
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


function strip0x(str: string) {
  return str.startsWith("0x") ? str.substring(2) : str;
}

function sha256sum(buff: Buffer) {
  return crypto.createHash("sha256").update(buff).digest("hex");
}

run().then(() => console.log("Done! " + processName));
