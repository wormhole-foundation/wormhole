import { RelayProviderProxy__factory } from "../../../ethers-contracts";
import { RelayProviderSetup__factory } from "../../../ethers-contracts";
import { RelayProviderImplementation__factory } from "../../../ethers-contracts";
import { MockRelayerIntegration__factory } from "../../../ethers-contracts";
import { CoreRelayerProxy__factory } from "../../../ethers-contracts";
import { CoreRelayerSetup__factory } from "../../../ethers-contracts";
import { CoreRelayerImplementation__factory } from "../../../ethers-contracts";

import {
  init,
  loadChains,
  loadPrivateKey,
  writeOutputFiles,
  ChainInfo,
  Deployment,
  getSigner,
  getCoreRelayerAddress,
  getCreate2Factory,
  getCoreRelayer,
  fetchSetupAddressCreate2,
} from "./env";
import { ethers } from "ethers";
import {
  Create2Factory__factory,
  ForwardWrapper__factory,
} from "../../../ethers-contracts";
import { wait } from "./utils";

export const setupContractSalt = Buffer.from("0xSetup");
export const proxyContractSalt = Buffer.from("0xGenericRelayer");

export async function deployRelayProviderImplementation(
  chain: ChainInfo
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
  chain: ChainInfo
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
  relayProviderImplementationAddress: string
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
    await getCoreRelayerAddress(chain)
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

export async function deployForwardWrapper(
  chain: ChainInfo,
  coreRelayerProxyAddress: string
): Promise<Deployment> {
  console.log("deployCoreRelayerLibrary " + chain.chainId);

  const result = await new ForwardWrapper__factory(getSigner(chain))
    .deploy(coreRelayerProxyAddress, chain.wormholeAddress)
    .then(deployed);
  console.log("Successfully deployed contract at " + result.address);
  return { address: result.address, chainId: chain.chainId };
}

export async function deployCoreRelayerImplementation(
  chain: ChainInfo,
  forwardWrapperAddress: string
): Promise<Deployment> {
  console.log("deployCoreRelayerImplementation " + chain.chainId);

  const result = await new CoreRelayerImplementation__factory(getSigner(chain))
    .deploy(forwardWrapperAddress)
    .then(deployed);

  console.log("Successfully deployed contract at " + result.address);
  return { address: result.address, chainId: chain.chainId };
}
export async function deployCoreRelayerSetup(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployCoreRelayerSetup " + chain.chainId);

  const create2Factory = getCreate2Factory(chain);
  const rx = await create2Factory
    .create2(setupContractSalt, CoreRelayerSetup__factory.bytecode)
    .then(wait);
  const address = Create2Factory__factory.createInterface().parseLog(rx.logs[0])
    .args[0];

  console.log("Successfully deployed contract at " + address);
  return { address, chainId: chain.chainId };
}

export async function deployCoreRelayerProxy(
  chain: ChainInfo,
  coreRelayerSetupAddress: string,
  coreRelayerImplementationAddress: string,
  wormholeAddress: string,
  relayProviderProxyAddress: string
): Promise<Deployment> {
  console.log("deployCoreRelayerProxy " + chain.chainId);

  const create2Factory = getCreate2Factory(chain);
  const expectedSetupAddr = await fetchSetupAddressCreate2(
    chain,
    create2Factory
  );
  if (coreRelayerSetupAddress !== expectedSetupAddr) {
    throw new Error(
      `coreRelayerSetupAddress different than expected. Expected: ${expectedSetupAddr} Actual: ${coreRelayerSetupAddress}`
    );
  }

  // deploy proxy and point at setup contract
  const data = new CoreRelayerProxy__factory().getDeployTransaction(
    coreRelayerSetupAddress
  ).data!;
  const rx = await create2Factory.create2(proxyContractSalt, data).then(wait);

  // call setup
  const governanceChainId = 1;
  const governanceContract =
    "0x0000000000000000000000000000000000000000000000000000000000000004";
  const proxy = CoreRelayerSetup__factory.connect(
    await getCoreRelayerAddress(chain, true),
    getSigner(chain)
  );
  await proxy
    .setup(
      coreRelayerImplementationAddress,
      chain.chainId,
      wormholeAddress,
      relayProviderProxyAddress,
      governanceChainId,
      governanceContract,
      chain.evmNetworkId
    )
    .then(wait);
  console.log("Successfully deployed contract at " + proxy.address);
  return { address: proxy.address, chainId: chain.chainId };
}

function link(bytecode: string, libName: String, libAddress: string) {
  //This doesn't handle the libName, because Forge embed a psuedonym into the bytecode, like
  //__$a7dd444e34bd28bbe3641e0101a6826fa7$__
  //This means we can't link more than one library per bytecode
  //const example = "__$a7dd444e34bd28bbe3641e0101a6826fa7$__"
  let symbol = /__.*?__/g;
  return bytecode.replace(symbol, libAddress.toLowerCase().substr(2));
}

const deployed = (x: ethers.Contract) => x.deployed();
