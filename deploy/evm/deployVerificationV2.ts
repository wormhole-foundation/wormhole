import { readFile } from "fs/promises";
import { createWalletClient, defineChain, http, isHex } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import yargs from "yargs";
import { hideBin } from 'yargs/helpers';
import { getContracts, toChain, UniversalAddress } from "@wormhole-foundation/sdk";

import { waitForTransactionReceipt } from "viem/actions";
import { EvmSerializableDeployment, saveDeployments } from "./deploymentArtifacts.js";
// TODO: replace this with readFile + JSON.parse
import compilerOutput from "../../verifiable-evm-build/WormholeVerifier.output.json" with {type: "json"};

// ICoreBridge coreBridge,
// uint32 initialMultisigKeyCount,
// uint32 initialSchnorrKeyCount,
// uint32 initialMultisigKeyPullLimit,
// bytes memory appendSchnorrKeyVaa

// export const wagmiAbi = [
//   ...
//   {
//     inputs: [{ name: "x", type: "uint32" }],
//     stateMutability: "nonpayable",
//     type: "constructor",
//   },
//   ...
// ] as const;

interface ChainDescriptor {
  description: string;
  eip155ChainId: number;
  chainId: number;
  rpc: string;
  coreV1Address: string;
}

interface Config {
  guardianSet: {
    initialMultisigKeyCount: number;
    initialSchnorrKeyCount: number;
    initialMultisigKeyPullLimit: number;
    appendSchnorrKeyVaa: string;
  };
  evm: {
    compilerOutput: string;
    chains: ChainDescriptor[]
  }
}


async function main() {
  const parser = yargs(hideBin(process.argv))
    .option('configFile', {
      description: 'Path to JSON config file.',
      demandOption: true,
      type: 'string',
    })
    .option('signer', {
      description: 'Path to JSON file that contains a signer private key in hex',
      demandOption: true,
      type: 'string',
    });
  const args = await parser.parse();
  const network = args.configFile.toLowerCase().includes("mainnet") ? "Mainnet" : "Testnet";

  const signerFile = await readFile(args.signer, "utf8");
  const signer = JSON.parse(signerFile);
  if (typeof signer !== "string" || !isHex(signer)) throw new Error("Unexpected signer file format.");

  const account = privateKeyToAccount(signer);
  const contractOutput = compilerOutput.contracts["src/evm/WormholeVerifier.sol"].WormholeVerifier;
  const abi = contractOutput.abi;
  const bytecode = `0x${contractOutput.evm.bytecode.object}`;
  if (typeof bytecode !== "string" || !isHex(bytecode)) throw new Error("Unexpected bytecode format.");

  const configFile = await readFile(args.configFile, "utf8");
  const config = JSON.parse(configFile) as Config;

  const tasks = await Promise.allSettled(config.evm.chains.map(async (chain) => {
    const chainName = toChain(chain.chainId) || chain.description;
    const viemChain = defineChain({
      id: chain.eip155ChainId,
      name: chainName,
      nativeCurrency: {
        decimals: 18,
        name: `(${chainName} gas token)`,
        symbol: `(${chainName} gas token symbol)`,
      },
      rpcUrls: {
        default: {
          http: [chain.rpc],
        },
      },
    });

    const walletClient = createWalletClient({
      chain: viemChain,
      transport: http(chain.rpc),
    });

    const coreV1Address = getCoreV1Address(chain, network)

    const constructorArgs = [
      coreV1Address,
      config.guardianSet.initialMultisigKeyCount,
      config.guardianSet.initialSchnorrKeyCount,
      config.guardianSet.initialMultisigKeyPullLimit,
      config.guardianSet.appendSchnorrKeyVaa,
    ];
    const txid = await walletClient.deployContract({
      abi,
      account,
      args: constructorArgs,
      bytecode,
    });

    const receipt = await waitForTransactionReceipt(walletClient, {
      hash: txid,
    });

    if (receipt.status !== "success") throw new Error(`Deploy tx failed in chain ${chainName}`);

    return {
      address: receipt.contractAddress!,
      chainId: chain.chainId as any,
      constructorArgs,
      deployTxid: txid,
    } satisfies EvmSerializableDeployment;
  }));

  const newDeployments = [];
  for (let i = 0; i < tasks.length; ++i) {
    const task = tasks[i];
    if (task.status === "fulfilled") {
      newDeployments.push(task.value);
    } else {
      console.error(`Failed deployment in chain ${config.evm.chains[i].description}:\n${task.reason}`);
    }
  }

  saveDeployments({verificationV2: newDeployments}, "deployVerificationV2", network);
}

function getCoreV1Address(chainDescriptor: ChainDescriptor, network: "Mainnet" | "Testnet") {
  const chainName = toChain(chainDescriptor.chainId);
  if (chainName === undefined) {
    return checkConfigCoreV1Address(chainDescriptor);
  }

  const sdkCoreAddress = getContracts(network, chainName).coreBridge;
  if (sdkCoreAddress === undefined) {
    return checkConfigCoreV1Address(chainDescriptor);
  }

  if (chainDescriptor.coreV1Address === undefined) {
    return sdkCoreAddress;
  }

  const configCoreAddress = new UniversalAddress(chainDescriptor.coreV1Address);
  if (!configCoreAddress.equals(new UniversalAddress(sdkCoreAddress))) {
    throw new Error(`Expected core v1 address to be ${sdkCoreAddress} but it's set to ${configCoreAddress.toNative(chainName).toString()} in the configuration`);
  }
  return sdkCoreAddress;
}

function checkConfigCoreV1Address(chainDescriptor: ChainDescriptor) {
  if (chainDescriptor.coreV1Address !== undefined) {
    console.error(`Warning: SDK core v1 address for chain ${chainDescriptor.chainId} is undefined.`);
    return chainDescriptor.coreV1Address;
  }

  throw new Error(`Missing core v1 address for chain ${chainDescriptor.chainId}`);
}

main().catch((error) => {
  console.error(error?.stack || error);
  process.exit(1);
});