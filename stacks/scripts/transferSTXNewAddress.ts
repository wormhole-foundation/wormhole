import { getStxAddress, randomSeedPhrase } from "@stacks/wallet-sdk"
import { HDKey } from "@scure/bip32";
import { deriveWalletKeys } from "@stacks/wallet-sdk";
import { DerivationType } from "@stacks/wallet-sdk";
import { deriveAccount } from "@stacks/wallet-sdk";
import { mnemonicToSeed } from "@scure/bip39";
import { StacksNetworks, type StacksNetworkName } from "@stacks/network";
import { getKeys } from "./utils";
import { broadcastTransaction, fetchNonce, makeSTXTokenTransfer } from "@stacks/transactions";

async function main() {

  const mnemonic = await randomSeedPhrase()
  const rootPrivateKey = await mnemonicToSeed(mnemonic)
  
  const rootNode1 = HDKey.fromMasterSeed(rootPrivateKey)
  const derived = await deriveWalletKeys(rootNode1 as any)
  const rootNode = HDKey.fromExtendedKey(derived.rootKey)
  const account = deriveAccount({
    rootNode: rootNode as any,
    index: 0,
    salt: derived.salt,
    stxDerivationType: DerivationType.Wallet,
  })
  const address = getStxAddress({ account, network: "devnet" })
  

  console.log(`Address: ${address} | Private Key: ${account.stxPrivateKey}`)

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
    const deployerBalance = await getPrincipalStxBalance(STACKS_API_URL, deployerAddress)
    console.log(`Deployer balance: ${deployerBalance}`)
    const recipientBalance = await getPrincipalStxBalance(STACKS_API_URL, address)
    console.log(`Recipient balance: ${recipientBalance}`)

    const transferAmount = 50_000_000n // 50 STX in microSTX
    
    let nonce = await fetchNonce({
      address: deployerAddress,
      client: { baseUrl: STACKS_API_URL },
    })

    console.log(`Transferring ${transferAmount} STX from: ${deployerAddress} (nonce: ${nonce}) to: ${address} in ${NETWORK_NAME} via ${STACKS_API_URL}`)

    const transaction = await makeSTXTokenTransfer({
      senderKey: deployerPrivateKey,
      recipient: address,
      amount: transferAmount,
      network: NETWORK_NAME,
      nonce: nonce,
      fee: 200_000n, // Fee in microSTX (0.2 STX)
      client: { baseUrl: STACKS_API_URL },
    })

    const broadcasted = await broadcastTransaction({
      transaction,
      network: NETWORK_NAME,  
      client: { baseUrl: STACKS_API_URL },
    })
    console.log(broadcasted)
}

main()

async function getPrincipalStxBalance(stacksApiUrl: string, principal: string): Promise<number> {
  const balance = await fetch(`${stacksApiUrl}/extended/v1/address/${principal}/balances`)
  return Number((await balance.json()).stx.balance) / 1e6
}
