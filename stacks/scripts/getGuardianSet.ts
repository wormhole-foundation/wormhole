import { StacksNetworks, type StacksNetworkName } from "@stacks/network"
import { getKeys } from "./utils"
import { cvToValue, fetchCallReadOnlyFunction } from "@stacks/transactions"
import { keccak256 } from "@wormhole-foundation/sdk-definitions"

(async() => {
  const STACKS_API_URL = process.env.STACKS_API_URL

  if(!STACKS_API_URL) {
    throw new Error("RPC_URL is required")
  }

  const DEPLOYER_MNEMONIC = process.env.DEPLOYER_MNEMONIC
  const DEPLOYER_PRIVATE_KEY = process.env.DEPLOYER_PRIVATE_KEY

  if((!!DEPLOYER_MNEMONIC && !!DEPLOYER_PRIVATE_KEY) || (!DEPLOYER_MNEMONIC && !DEPLOYER_PRIVATE_KEY)) {
    throw new Error("Only one of DEPLOYER_MNEMONIC or DEPLOYER_PRIVATE_KEY must be set")
  }

  const NETWORK_NAME = process.env.NETWORK_NAME as StacksNetworkName

  if(!NETWORK_NAME) {
    throw new Error("NETWORK_NAME is required")
  }

  if (!StacksNetworks.includes(NETWORK_NAME)) {
    throw new Error(`Invalid NETWORK_NAME: ${NETWORK_NAME} | Valid networks: ${StacksNetworks.join(", ")}`)
  }

  const {privateKey: deployerPrivateKey, address: deployerAddress} = await getKeys(NETWORK_NAME, DEPLOYER_MNEMONIC, DEPLOYER_PRIVATE_KEY)

  const coreContractName = "wormhole-core-v4"

  const guardianSet = await fetchCallReadOnlyFunction({
    contractAddress: deployerAddress,
    contractName: coreContractName,
    functionName: "get-active-guardian-set",
    functionArgs: [],
    senderAddress: deployerAddress,
    network: NETWORK_NAME,
    client: { baseUrl: STACKS_API_URL },
  })

  const guardianSetValue = cvToValue(guardianSet)
  const guardians = guardianSetValue.value.guardians.value
  console.log(`Number of guardians: ${guardians.length}`)
  guardians.forEach((guardian: any, index: number) => {
    const uncompressedPubKey = guardian.value["uncompressed-public-key"].value
    const compressedPubKey = guardian.value["compressed-public-key"].value
    
    const hash = keccak256(Buffer.from(uncompressedPubKey.slice(2), 'hex'))
    const ethereumAddress = "0x" + hash.toHex().slice(-40)
    
    console.log(`\t [Guardian ${index}]:`)
    console.log(`\t \t Ethereum Address:`, ethereumAddress)
    console.log(`\t \t Compressed public key:`, compressedPubKey)
    console.log(`\t \t Uncompressed public key:`, uncompressedPubKey)
  })
})()
