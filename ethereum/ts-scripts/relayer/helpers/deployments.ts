import { RelayProviderProxy__factory } from "../../../ethers-contracts";
import { RelayProviderSetup__factory } from "../../../ethers-contracts";
import { RelayProviderImplementation__factory } from "../../../ethers-contracts";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";
import { CoreRelayer__factory } from "../../../ethers-contracts";

import {
  ChainInfo,
  Deployment,
  getSigner,
  getCoreRelayerAddress,
  getCreate2Factory,
} from "./env";
import { ethers } from "ethers";
import {
  Create2Factory__factory,
} from "../../../ethers-contracts";
import { wait } from "./utils";

export const setupContractSalt = Buffer.from("0xSetup");
export const proxyContractSalt = Buffer.from("0xGenericRelayer");

export async function deployRelayProviderImplementation(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployRelayProviderImplementation " + chain.chainId);
  const signer = getSigner(chain);

  const contractInterface = RelayProviderImplementation__factory.createInterface();
  const bytecode = RelayProviderImplementation__factory.bytecode;
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

export async function deployRelayProviderSetup(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployRelayProviderSetup " + chain.chainId);
  const signer = getSigner(chain);
  const contractInterface = RelayProviderSetup__factory.createInterface();
  const bytecode = RelayProviderSetup__factory.bytecode;
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
export async function deployRelayProviderProxy(
  chain: ChainInfo,
  relayProviderSetupAddress: string,
  relayProviderImplementationAddress: string,
): Promise<Deployment> {
  console.log("deployRelayProviderProxy " + chain.chainId);

  const signer = getSigner(chain);
  const contractInterface = RelayProviderProxy__factory.createInterface();
  const bytecode = RelayProviderProxy__factory.bytecode;
  //@ts-ignore
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    signer
  );

  let ABI = ["function setup(address,uint16)"];
  let iface = new ethers.utils.Interface(ABI);
  let encodedData = iface.encodeFunctionData("setup", [
    relayProviderImplementationAddress,
    chain.chainId,
  ]);

  const contract = await factory.deploy(relayProviderSetupAddress, encodedData);
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}

export async function deployMockIntegration(
  chain: ChainInfo,
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
    await getCoreRelayerAddress(chain)
  );
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address);
    return { address: result.address, chainId: chain.chainId };
  });
}

export async function deployCreate2Factory(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployCreate2Factory " + chain.chainId);

  const result = await new Create2Factory__factory(getSigner(chain))
    .deploy()
    .then(deployed);
  console.log(`Successfully deployed contract at ${result.address}`);
  return { address: result.address, chainId: chain.chainId };
}

export async function deployCoreRelayerImplementation(
  chain: ChainInfo,
): Promise<Deployment> {
  console.log("deployCoreRelayerImplementation " + chain.chainId);

  const result = await new CoreRelayer__factory(getSigner(chain))
    .deploy(chain.wormholeAddress)
    .then(deployed);

  console.log("Successfully deployed contract at " + result.address);
  return { address: result.address, chainId: chain.chainId };
}

export async function deployCoreRelayerProxy(
  chain: ChainInfo,
  coreRelayerImplementationAddress: string,
  defaultRelayProvider: string,
): Promise<Deployment> {
  console.log("deployCoreRelayerProxy " + chain.chainId);

  const create2Factory = getCreate2Factory(chain);

  const initData = CoreRelayer__factory.createInterface().encodeFunctionData(
    "initialize",
    [ethers.utils.getAddress(defaultRelayProvider)]
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
