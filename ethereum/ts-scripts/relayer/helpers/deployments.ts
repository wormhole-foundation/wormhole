import {
  DeliveryProviderProxy__factory,
  DeliveryProviderSetup__factory,
  DeliveryProviderImplementation__factory,
  MockRelayerIntegration__factory,
  WormholeRelayer__factory,
  Create2Factory__factory,
} from "../../../ethers-contracts";

import {
  ChainInfo,
  Deployment,
  getSigner,
  getWormholeRelayerAddress,
  getCreate2Factory,
  env,
  getChain,
  getProvider,
} from "./env";
import { ethers } from "ethers";
import { wait } from "./utils";
import { CONTRACTS, ChainId, coalesceChainName } from "@certusone/wormhole-sdk";

export const setupContractSalt = Buffer.from("0xSetup");
export const proxyContractSalt = Buffer.from("0xGenericRelayer");

export async function deployDeliveryProviderImplementation(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployDeliveryProviderImplementation " + chain.chainId);
  const signer = await getSigner(chain);
  const factory = new DeliveryProviderImplementation__factory(signer);

  const overrides = await buildOverridesDeploy(factory, chain, []);
  const contract = await factory.deploy(overrides);
  const receipt = await contract.deployTransaction.wait();
  console.log("Successfully deployed contract at " + receipt.contractAddress);
  return { address: receipt.contractAddress, chainId: chain.chainId };
}

export async function deployDeliveryProviderSetup(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployDeliveryProviderSetup " + chain.chainId);

  const signer = await getSigner(chain);
  const factory = new DeliveryProviderSetup__factory(signer);

  const overrides = await buildOverridesDeploy(factory, chain, []);
  const contract = await factory.deploy(overrides);
  const receipt = await contract.deployTransaction.wait();
  console.log("Successfully deployed contract at " + receipt.contractAddress);
  return { address: receipt.contractAddress, chainId: chain.chainId };
}

/**
 * Deploys `DeliveryProvider` proxy with the old (account, nonce) tuple hashing creation mechanism.
 */
export async function deployDeliveryProviderProxy(
  chain: ChainInfo,
  deliveryProviderSetupAddress: string,
  deliveryProviderImplementationAddress: string,
): Promise<Deployment> {
  console.log("deployDeliveryProviderProxy " + chain.chainId);

  const signer = await getSigner(chain);
  const factory = new DeliveryProviderProxy__factory(signer);

  const setupInterface = DeliveryProviderSetup__factory.createInterface();
  const encodedData = setupInterface.encodeFunctionData("setup", [
    deliveryProviderImplementationAddress,
    chain.chainId,
  ]);

  const overrides = await buildOverridesDeploy(factory, chain, [
    deliveryProviderSetupAddress,
    encodedData,
  ]);
  const contract = await factory.deploy(
    deliveryProviderSetupAddress,
    encodedData,
    overrides,
  );
  const receipt = await contract.deployTransaction.wait();
  console.log("Successfully deployed contract at " + receipt.contractAddress);
  return { address: receipt.contractAddress, chainId: chain.chainId };
}

export async function deployMockIntegration(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployMockIntegration " + chain.chainId);

  const signer = await getSigner(chain);
  const factory = new MockRelayerIntegration__factory(signer);

  const wormholeRelayerAddress = await getWormholeRelayerAddress(chain);
  checkCoreAddress(chain.wormholeAddress, env, chain.chainId);
  const overrides = await buildOverridesDeploy(factory, chain, [
    chain.wormholeAddress,
    wormholeRelayerAddress,
  ]);
  const contract = await factory.deploy(
    chain.wormholeAddress,
    wormholeRelayerAddress,
    overrides,
  );
  const receipt = await contract.deployTransaction.wait();
  console.log("Successfully deployed contract at " + receipt.contractAddress);
  return { address: receipt.contractAddress, chainId: chain.chainId };
}

let ethC2Promise: Promise<string>;

/**
 * Deploys `Create2Factory` with the old (account, nonce) tuple hashing creation mechanism.
 * To achieve same address multichain deployments, ensure that the
 * same (address, nonce) tx pair creates the factory across all target chains.
 */
export async function deployCreate2Factory(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployCreate2Factory " + chain.chainId);

  const signer = await getSigner(chain);
  const factory = new Create2Factory__factory(signer);

  const signerAddress = await signer.getAddress();
  const ethChain = getChain(2);
  const ethChainProvider = getProvider(ethChain);
  const ethNetwork = await ethChainProvider.getNetwork();
  if (ethNetwork.chainId === 1 && signerAddress.toLowerCase() === "0x5623bdf52b51085c807a5dc39152eed05825f5fd") {
    // Here we check that the bytecode matches against Ethereum.
    if (ethC2Promise === undefined) {
      // we assign the promise immediately to avoid race conditions
      ethC2Promise = (async () => {
        const ethFactory = await getCreate2Factory(ethChain, ethChainProvider);
        return ethFactory.provider.getCode(ethFactory.address);
      })();
    }

    const ethCreate2FactoryCodeStr = strip0x(await ethC2Promise);
    // Note that we're looking up the deployed bytecode within the "init" bytecode,
    // i.e. bytecode that contains both the constructor and the object to be deployed.

    // The create 2 factory contract has these immutables (taken from the compiler output for Create2Factory):
    // "immutableReferences": {
    //   "20558": [
    //     {
    //       "start": 704,
    //       "length": 32
    //     },
    //     {
    //       "start": 1007,
    //       "length": 32
    //     }
    //   ],
    //   "20560": [
    //     {
    //       "start": 1134,
    //       "length": 32
    //     }
    //   ]
    // }
    // These are set as soon as the constructor executes and they only depend on the account where the constructor executes.
    // The constructor has no parameters.
    // Thus, we'll just zero out these locations in the ethereum bytecode so that we can look up the deployed bytecode within the deployable bytecode that we are about to use.
    const ethCreate2FactoryCode = Buffer.from(ethCreate2FactoryCodeStr, "hex");
    const zeroWord = Buffer.alloc(32, 0);
    ethCreate2FactoryCode.set(zeroWord, 704);
    ethCreate2FactoryCode.set(zeroWord, 1007);
    ethCreate2FactoryCode.set(zeroWord, 1134);
    const comparableEthCode = ethCreate2FactoryCode.toString("hex");

    if (!factory.bytecode.includes(comparableEthCode)) {
      throw new Error("Factory contract bytecode doesn't match the factory deployed in Ethereum. Aborting deployment.");
    }
  }

  const overrides = await buildOverridesDeploy(factory, chain, []);
  const contract = await factory.deploy(overrides).then(deployed);
  console.log(`Successfully deployed contract at ${contract.address}`);
  return { address: contract.address, chainId: chain.chainId };
}

function checkCoreAddress(wormhole: string, env: string, chainId: ChainId) {
  const chainName = coalesceChainName(chainId);
  if (chainName === undefined) {
    return;
  }

  // We assume other environments are local devnets
  const contractSet = env === "mainnet" ? "MAINNET" : env === "testnet" ? "TESTNET" : undefined;
  if (contractSet === undefined) return;

  const sdkWormhole = CONTRACTS[contractSet][chainName].core;
  if (sdkWormhole === undefined) {
    console.error(`Warning: SDK Wormhole address for chain ${chainId} is undefined.`);
    return;
  }
  if (sdkWormhole.toLowerCase() !== wormhole.toLowerCase()) {
    throw new Error(`Expected wormhole address to be ${sdkWormhole} but it's set to ${wormhole} in chains.json`);
  }
}

export async function deployWormholeRelayerImplementation(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployWormholeRelayerImplementation " + chain.chainId);

  const signer = await getSigner(chain);
  const factory = new WormholeRelayer__factory(signer);

  checkCoreAddress(chain.wormholeAddress, env, chain.chainId);
  const overrides = await buildOverridesDeploy(factory, chain, [
    chain.wormholeAddress,
  ]);
  const result = await factory
    .deploy(chain.wormholeAddress, overrides)
    .then(deployed);

  console.log(
    `Successfully deployed WormholeRelayer contract at ${result.address}`,
  );
  return { address: result.address, chainId: chain.chainId };
}

/**
 * Deploys `WormholeRelayer` proxy with the CREATE2 factory.
 */
export async function deployWormholeRelayerProxy(
  chain: ChainInfo,
  coreRelayerImplementationAddress: string,
  defaultDeliveryProvider: string,
): Promise<Deployment> {
  console.log("deployWormholeRelayerProxy " + chain.chainId);

  const create2Factory = await getCreate2Factory(chain);

  const initData = WormholeRelayer__factory.createInterface().encodeFunctionData(
    "initialize",
    [ethers.utils.getAddress(defaultDeliveryProvider)],
  );
  const overrides = await buildOverrides(
    () =>
      create2Factory.estimateGas.create2Proxy(
        proxyContractSalt,
        coreRelayerImplementationAddress,
        initData,
      ),
    chain,
  );
  const rx = await create2Factory
    .create2Proxy(
      proxyContractSalt,
      coreRelayerImplementationAddress,
      initData,
      overrides,
    )
    .then(wait);

  let proxyAddress: string;
  // pull proxyAddress from create2Factory logs
  for (const log of rx.logs) {
    try {
      if (log.address == create2Factory.address) {
        proxyAddress = create2Factory.interface.parseLog(log).args.addr;
      }
    } catch (e) {}
  }
  const signer = await getSigner(chain);
  const computedAddr = await create2Factory.computeProxyAddress(
    await signer.getAddress(),
    proxyContractSalt,
  );
  if (proxyAddress! !== computedAddr) {
    console.error("Computed address does not match desired");
  }

  console.log(`Successfully deployed contract WormholeRelayerProxy at ${computedAddr}`);
  return { address: computedAddr, chainId: chain.chainId };
}

const deployed = (x: ethers.Contract) => x.deployed();

const estimateGasDeploy = async (
  factory: ethers.ContractFactory,
  args: unknown[],
): Promise<ethers.BigNumber> => {
  const deployTxArgs = factory.getDeployTransaction(...args);
  return factory.signer.estimateGas(deployTxArgs);
};

const buildOverridesDeploy = async (
  factory: ethers.ContractFactory,
  chain: ChainInfo,
  args: unknown[],
): Promise<ethers.Overrides> => {
  return buildOverrides(() => estimateGasDeploy(factory, args), chain);
};

async function overshootEstimationGas(
  estimate: () => Promise<ethers.BigNumber>,
): Promise<ethers.BigNumber> {
  const gasEstimate = await estimate();
  // we multiply gas estimation by a factor 1.1 to avoid slightly skewed estimations from breaking transactions.
  return gasEstimate.mul(1100).div(1000);
}

export async function buildOverrides(
  estimate: () => Promise<ethers.BigNumber>,
  chain: ChainInfo,
): Promise<ethers.Overrides> {
  const overrides: ethers.Overrides = {
    gasLimit: await overshootEstimationGas(estimate),
  };
  // If this is Polygon or Fantom, use the legacy tx envelope to avoid bad gas price feeds.
  if (chain.chainId === 5 || chain.chainId === 10) {
    overrides.type = 0;
  } else if (chain.chainId === 4) {
    // This is normally autodetected in bsc but we want to set the gas price to a fixed value.
    // We need to ensure we are using the correct tx envelope in that case.
    overrides.type = 0;
    overrides.gasPrice = ethers.utils.parseUnits("1", "gwei");
  } else if (chain.chainId === 23) {
    // Arbitrum gas price feeds are excessive on public endpoints too apparently.
    overrides.type = 2;
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.3", "gwei");
    overrides.maxPriorityFeePerGas = 0;
  } else if (chain.chainId === 34) {
    overrides.type = 0;
  } else if (chain.chainId === 36) {
    overrides.type = 2;
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.005", "gwei");
    overrides.maxPriorityFeePerGas = ethers.utils.parseUnits("0.001", "gwei");
  }
  return overrides;
}

function strip0x(str: string) {
  return str.startsWith("0x") ? str.substring(2) : str;
}