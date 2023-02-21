import { RelayProviderProxy__factory } from "../../../sdk/src/ethers-contracts/factories/RelayProviderProxy__factory"
import { RelayProviderSetup__factory } from "../../../sdk/src/ethers-contracts/factories/RelayProviderSetup__factory"
import { RelayProviderImplementation__factory } from "../../../sdk/src/ethers-contracts/factories/RelayProviderImplementation__factory"
import { MockRelayerIntegration__factory } from "../../../sdk/src"
import { CoreRelayerProxy__factory } from "../../../sdk/src/ethers-contracts/factories/CoreRelayerProxy__factory"
import { CoreRelayerSetup__factory } from "../../../sdk/src/ethers-contracts/factories/CoreRelayerSetup__factory"
import { CoreRelayerImplementation__factory } from "../../../sdk/src/ethers-contracts/factories/CoreRelayerImplementation__factory"
import { CoreRelayerLibrary__factory } from "../../../sdk/src/ethers-contracts/factories/CoreRelayerLibrary__factory"

import {
  init,
  loadChains,
  loadPrivateKey,
  writeOutputFiles,
  ChainInfo,
  Deployment,
  getSigner,
  getCoreRelayerAddress,
} from "./env"
import { ethers } from "ethers"

export async function deployRelayProviderImplementation(
  chain: ChainInfo
): Promise<Deployment> {
  console.log("deployRelayProviderImplementation " + chain.chainId)
  const signer = getSigner(chain)

  const contractInterface = RelayProviderImplementation__factory.createInterface()
  const bytecode = RelayProviderImplementation__factory.bytecode
  //@ts-ignore
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)
  const contract = await factory.deploy()
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}

export async function deployRelayProviderSetup(chain: ChainInfo): Promise<Deployment> {
  console.log("deployRelayProviderSetup " + chain.chainId)
  const signer = getSigner(chain)
  const contractInterface = RelayProviderSetup__factory.createInterface()
  const bytecode = RelayProviderSetup__factory.bytecode
  //@ts-ignore
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)
  const contract = await factory.deploy()
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}
export async function deployRelayProviderProxy(
  chain: ChainInfo,
  relayProviderSetupAddress: string,
  relayProviderImplementationAddress: string
): Promise<Deployment> {
  console.log("deployRelayProviderProxy " + chain.chainId)

  const signer = getSigner(chain)
  const contractInterface = RelayProviderProxy__factory.createInterface()
  const bytecode = RelayProviderProxy__factory.bytecode
  //@ts-ignore
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)

  let ABI = ["function setup(address,uint16)"]
  let iface = new ethers.utils.Interface(ABI)
  let encodedData = iface.encodeFunctionData("setup", [
    relayProviderImplementationAddress,
    chain.chainId,
  ])

  const contract = await factory.deploy(relayProviderSetupAddress, encodedData)
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}

export async function deployMockIntegration(chain: ChainInfo): Promise<Deployment> {
  console.log("deployMockIntegration " + chain.chainId)

  let signer = getSigner(chain)
  const contractInterface = MockRelayerIntegration__factory.createInterface()
  const bytecode = MockRelayerIntegration__factory.bytecode
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)
  const contract = await factory.deploy(
    chain.wormholeAddress,
    getCoreRelayerAddress(chain)
  )
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}

export async function deployCoreRelayerLibrary(chain: ChainInfo): Promise<Deployment> {
  console.log("deployCoreRelayerLibrary " + chain.chainId)

  let signer = getSigner(chain)
  const contractInterface = CoreRelayerLibrary__factory.createInterface()
  const bytecode = CoreRelayerLibrary__factory.bytecode
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)
  const contract = await factory.deploy()
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}

export async function deployCoreRelayerImplementation(
  chain: ChainInfo,
  coreRelayerLibraryAddress: string
): Promise<Deployment> {
  console.log("deployCoreRelayerImplementation " + chain.chainId)
  const signer = getSigner(chain)
  const contractInterface = CoreRelayerImplementation__factory.createInterface()
  const bytecode: string = CoreRelayerImplementation__factory.bytecode

  /*
  Linked libraries in EVM are contained in the bytecode and linked at compile time.
  However, the linked address of the CoreRelayerLibrary is not known until deployment time,
  So, rather that recompiling the contracts with a static link, we modify the bytecode directly 
  once we have the CoreRelayLibraryAddress.
  */
  const bytecodeWithLibraryLink = link(
    bytecode,
    "CoreRelayerLibrary",
    coreRelayerLibraryAddress
  )

  //@ts-ignore
  const factory = new ethers.ContractFactory(
    contractInterface,
    bytecodeWithLibraryLink,
    signer
  )
  const contract = await factory.deploy()
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}
export async function deployCoreRelayerSetup(chain: ChainInfo): Promise<Deployment> {
  console.log("deployCoreRelayerSetup " + chain.chainId)
  const signer = getSigner(chain)
  const contractInterface = CoreRelayerSetup__factory.createInterface()
  const bytecode = CoreRelayerSetup__factory.bytecode
  //@ts-ignore
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)
  const contract = await factory.deploy()
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}
export async function deployCoreRelayerProxy(
  chain: ChainInfo,
  coreRelayerSetupAddress: string,
  coreRelayerImplementationAddress: string,
  wormholeAddress: string,
  relayProviderProxyAddress: string
): Promise<Deployment> {
  console.log("deployCoreRelayerProxy " + chain.chainId)
  const signer = getSigner(chain)
  const contractInterface = CoreRelayerProxy__factory.createInterface()
  const bytecode = CoreRelayerProxy__factory.bytecode
  //@ts-ignore
  const factory = new ethers.ContractFactory(contractInterface, bytecode, signer)

  const governanceChainId = 1
  const governanceContract =
    "0x0000000000000000000000000000000000000000000000000000000000000004"

  let ABI = ["function setup(address,uint16,address,address,uint16,bytes32,uint256)"]
  let iface = new ethers.utils.Interface(ABI)
  let encodedData = iface.encodeFunctionData("setup", [
    coreRelayerImplementationAddress,
    chain.chainId,
    wormholeAddress,
    relayProviderProxyAddress,
    governanceChainId,
    governanceContract,
    chain.evmNetworkId,
  ])

  const contract = await factory.deploy(coreRelayerSetupAddress, encodedData)
  return await contract.deployed().then((result) => {
    console.log("Successfully deployed contract at " + result.address)
    return { address: result.address, chainId: chain.chainId }
  })
}
function link(bytecode: string, libName: String, libAddress: string) {
  //This doesn't handle the libName, because Forge embed a psuedonym into the bytecode, like
  //__$a7dd444e34bd28bbe3641e0101a6826fa7$__
  //This means we can't link more than one library per bytecode
  //const example = "__$a7dd444e34bd28bbe3641e0101a6826fa7$__"
  let symbol = /__.*?__/g
  return bytecode.replace(symbol, libAddress.toLowerCase().substr(2))
}
