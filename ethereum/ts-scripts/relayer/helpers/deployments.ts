import { DeliveryProviderProxy__factory } from "../../../ethers-contracts";
import { DeliveryProviderSetup__factory } from "../../../ethers-contracts";
import { DeliveryProviderImplementation__factory } from "../../../ethers-contracts";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";
import { WormholeRelayer__factory } from "../../../ethers-contracts";

import {
  ChainInfo,
  Deployment,
  getSigner,
  getWormholeRelayerAddress,
  getCreate2Factory,
} from "./env";
import { ethers } from "ethers";
import { Create2Factory__factory } from "../../../ethers-contracts";
import { wait } from "./utils";

export const setupContractSalt = Buffer.from("0xSetup");
export const proxyContractSalt = Buffer.from("0xGenericRelayer");

export async function deployDeliveryProviderImplementation(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployDeliveryProviderImplementation " + chain.chainId);
  const signer = getSigner(chain);

  const contractInterface = DeliveryProviderImplementation__factory.createInterface();
  const bytecode = DeliveryProviderImplementation__factory.bytecode;
  //@ts-ignore
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    signer
  );
  const contract = await factory.deploy();
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}

export async function deployDeliveryProviderSetup(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployDeliveryProviderSetup " + chain.chainId);
  const signer = getSigner(chain);
  const contractInterface = DeliveryProviderSetup__factory.createInterface();
  const bytecode = DeliveryProviderSetup__factory.bytecode;
  //@ts-ignore
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    signer
  );
  const contract = await factory.deploy();
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}
export async function deployDeliveryProviderProxy(
  chain: ChainInfo,
  deliveryProviderSetupAddress: string,
  deliveryProviderImplementationAddress: string
): Promise<Deployment> {
  console.log("deployDeliveryProviderProxy " + chain.chainId);

  const signer = getSigner(chain);
  const contractInterface = DeliveryProviderProxy__factory.createInterface();
  const bytecode = DeliveryProviderProxy__factory.bytecode;
  //@ts-ignore
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    signer
  );

  let ABI = ["function setup(address,uint16)"];
  let iface = new ethers.utils.Interface(ABI);
  let encodedData = iface.encodeFunctionData("setup", [
    deliveryProviderImplementationAddress,
    chain.chainId,
  ]);

  const contract = await factory.deploy(
    deliveryProviderSetupAddress,
    encodedData
  );
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}

export async function deployMockIntegration(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployMockIntegration " + chain.chainId);

  let signer = getSigner(chain);
  const contractInterface = MockRelayerIntegration__factory.createInterface();
  const bytecode = MockRelayerIntegration__factory.bytecode;
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    signer
  );
  const contract = await factory.deploy(
    chain.wormholeAddress,
    await getWormholeRelayerAddress(chain)
  );
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}

export async function deployCreate2Factory(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployCreate2Factory " + chain.chainId);

  const result = await new Create2Factory__factory(getSigner(chain))
    .deploy()
    .then(deployed);
  console.log(`Successfully deployed contract at ${result.address}`);
  return { address: result.address, chainId: chain.chainId };
}

export async function deployWormholeRelayerImplementation(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployWormholeRelayerImplementation " + chain.chainId);

  const result = await new WormholeRelayer__factory(getSigner(chain))
    .deploy(chain.wormholeAddress)
    .then(deployed);

  console.log("Successfully deployed contract at " + result.address);
  return { address: result.address, chainId: chain.chainId };
}

export async function deployWormholeRelayerProxy(
  chain: ChainInfo,
  coreRelayerImplementationAddress: string,
  defaultDeliveryProvider: string
): Promise<Deployment> {
  console.log("deployWormholeRelayerProxy " + chain.chainId);

  const create2Factory = getCreate2Factory(chain);

  const initData = WormholeRelayer__factory.createInterface().encodeFunctionData(
    "initialize",
    [ethers.utils.getAddress(defaultDeliveryProvider)]
  );
  const rx = await create2Factory
    .create2Proxy(proxyContractSalt, coreRelayerImplementationAddress, initData)
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
  const computedAddr = await create2Factory.computeProxyAddress(
    getSigner(chain).address,
    proxyContractSalt
  );
  if (proxyAddress! !== computedAddr) {
    console.error("Computed address does not match desired");
  }

  console.log("Successfully deployed contract at " + computedAddr);
  return { address: computedAddr, chainId: chain.chainId };
}

const deployed = (x: ethers.Contract) => x.deployed();
