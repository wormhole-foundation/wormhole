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
  getCreate2FactoryAddress,
} from "./env";
import { ethers } from "ethers";
import { wait } from "./utils";
import { ChainId, getContracts, toChain } from "@wormhole-foundation/sdk";

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

let ethC2Promise: Promise<Create2Factory__factory>;

/**
 * Deploys `Create2Factory` with the old (account, nonce) tuple hashing creation mechanism.
 * To achieve same address multichain deployments, ensure that the
 * same (address, nonce) tx pair creates the factory across all target chains.
 */
export async function deployCreate2Factory(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployCreate2Factory " + chain.chainId);

  let factory = new Create2Factory__factory();

  // This needs to be the Ethereum chain. We want to check whether we are on mainnet or not.
  let ethChain;
  try {
    ethChain = getChain(2);
  } catch {
    if (env.toLowerCase() === "mainnet") throw new Error(`This is a mainnet deployment but the Ethereum chain is not defined in chains.json`);
    console.log(`Couldn't retrieve the Ethereum chain. Make sure this is not a mainnet deployment.`);
  }
  if (ethChain !== undefined) {
    const ethChainProvider = getProvider(ethChain);
    const ethNetwork = await ethChainProvider.getNetwork();
    if (ethNetwork.chainId === 1) {
      console.log(`Retrieving Create2Factory from Ethereum`);
      // Here we fetch the creation bytecode from Ethereum.
      // We also perform a few sanity checks to ensure that the retrieved creation bytecode looks good:
      // 1. The transaction receipt should contain the expected address for the `Create2Factory`.
      // 2. The bytecode hash should be a specific one.
      //
      // Why do this? The creation bytecode of `SimpleProxy` is part of the hash function that derives the address.
      // This means that to reproduce the same address on a different chain, you need to create the exact same `SimpleProxy` contract.
      // Since the compiler inserts metadata hashes, and potentially other properties, that don't impact the functionality,
      // we reuse the same creation bytecode that we originally used instead of attempting to tune newer compiler versions to produce the same bytecode.
      if (ethC2Promise === undefined) {
        // We assign the promise immediately to avoid race conditions
        ethC2Promise = (async () => {
          const factoryCreationTxid = "0xfd6551a91a2e9f423285a2e86f7f480341a658dda1ff1d8bc9167b2b7ec77caa";
          const ethFactoryAddress = getCreate2FactoryAddress(ethChain);
          const factoryReceipt = await ethChainProvider.getTransactionReceipt(factoryCreationTxid);
          if (factoryReceipt.contractAddress !== ethFactoryAddress) {
            throw new Error("Wrong txid for the transaction that created the Create2Factory in Ethereum mainnet.");
          }
          const ethFactoryTx = await ethChainProvider.getTransaction(factoryCreationTxid);

          const expectedCreationCodeHash = "0x4b72c18c9a1a24d8406bde2edc283025bd33513d13c51601bb02dd4f298ada7d";
          const fetchedCreationCodeHash = ethers.utils.sha256(ethFactoryTx.data);
          if (expectedCreationCodeHash !== fetchedCreationCodeHash) {
            throw new Error(`Creation code mismatch for Create2Factory. Found: ${fetchedCreationCodeHash} Expected: ${expectedCreationCodeHash}`);
          }
          return new Create2Factory__factory(Create2Factory__factory.createInterface(), ethFactoryTx.data);
        })();
      }

      factory = await ethC2Promise;
    }
  }

  // We need to connect the signer here because we're overwriting the factory when deploying to mainnet.
  const signer = await getSigner(chain);
  factory = factory.connect(signer);
  const overrides = await buildOverridesDeploy(factory, chain, []);
  const contract = await factory.deploy(overrides).then(deployed);
  console.log(`Successfully deployed contract at ${contract.address}`);
  return { address: contract.address, chainId: chain.chainId };
}

function checkCoreAddress(wormhole: string, env: string, chainId: ChainId) {
  const chainName = toChain(chainId);
  if (chainName === undefined) {
    return;
  }

  // We assume other environments are local devnets
  const contractSet = env === "mainnet" ? "Mainnet" : env === "testnet" ? "Testnet" : undefined;
  if (contractSet === undefined) return;

  const sdkWormhole = getContracts(contractSet, chainName).coreBridge;
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

     // Use 5 gwei for testnet (chainId 97), 1 otherwise.
     overrides.gasPrice = ethers.utils.parseUnits(chain.evmNetworkId === 97 ? "5" : "1", "gwei");
  } else if (chain.chainId === 23) {
    // Arbitrum gas price feeds are excessive on public endpoints too apparently.
    overrides.type = 2;
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.3", "gwei");
    overrides.maxPriorityFeePerGas = 0;
  } else if (chain.chainId === 34) {
    overrides.type = 0;
  } else if (chain.chainId === 35) {
    overrides.type = 2;
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.1", "gwei");
    overrides.maxPriorityFeePerGas = 0;
  } else if (chain.chainId === 36) {
    overrides.type = 2;
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.08", "gwei");
    overrides.maxPriorityFeePerGas = ethers.utils.parseUnits("0.000000001", "gwei");
  } else if (chain.chainId === 37) {
    overrides.type = 0;
  } else if (chain.chainId === 45) {
    overrides.type = 2;
    overrides.maxPriorityFeePerGas = ethers.utils.parseUnits("0.0001", "gwei");
    overrides.maxFeePerGas = ethers.utils.parseUnits("0.001", "gwei");
  }
  return overrides;
}

function strip0x(str: string) {
  return str.startsWith("0x") ? str.substring(2) : str;
}