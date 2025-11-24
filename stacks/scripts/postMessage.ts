import { StacksNetworks, type StacksNetworkName } from "@stacks/network"
import { getKeys } from "./utils"
import { broadcastTransaction, Cl, makeContractCall } from "@stacks/transactions"

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
  console.log(`Deployer address: ${deployerAddress}`)
  const payload = `deadbeef`

  const postMessageTx = await makeContractCall({
    contractName: "wormhole-core-v4",
    contractAddress: deployerAddress,
    functionName: "post-message",
    functionArgs: [
      Cl.buffer(Uint8Array.from(Buffer.from(payload, "hex"))),
      Cl.uint(0),
      Cl.none()
    ],
    senderKey: deployerPrivateKey,
    network: NETWORK_NAME,
    client: { baseUrl: STACKS_API_URL },
    postConditionMode: "allow",
    fee: 6000000,
  })

  const response = await broadcastTransaction({
    transaction: postMessageTx,
    network: NETWORK_NAME,
    client: { baseUrl: STACKS_API_URL },
  })

  console.log(`Response:`, response)

})()
