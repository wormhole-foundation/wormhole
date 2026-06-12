import { ethers } from "ethers"
import { sleep } from "bun"
import { tokenBridge as tokenBridgeContracts } from '@wormhole-foundation/sdk-base/contracts';
import { chainIds, chainIdToChain, chainToPlatform, toUniversal } from "@wormhole-foundation/sdk";
import { TOKEN_BRIDGE_ABI } from "./abi";

async function main()  {

  const RPC = process.argv[2]
  const TOKEN_BRIDGE_ADDRESS = ethers.getAddress(process.argv[3])
  const TOKEN_BRIDGE_IMPLEMENTATION = ethers.getAddress(process.argv[4])
  const CORE_BRIDGE_ADDRESS = ethers.getAddress(process.argv[5])
  const WETH = ethers.getAddress(process.argv[6])
  const WORMHOLE_CHAIN_ID = parseInt(process.argv[7])
  const EVM_CHAIN_ID = parseInt(process.argv[8])
  
  /**
   * Misc/common configuration
  */
  const SLEEP_BETWEEN_RPC_CALLS_IN_MS = 0
  const GOVERNANCE_CHAIN_ID = 1
  const GOVERNANCE_CONTRACT = "0x0000000000000000000000000000000000000000000000000000000000000004"
  const FINALITY = 1

  const provider = new ethers.JsonRpcProvider(RPC)
  const tokenBridge = new ethers.Contract(TOKEN_BRIDGE_ADDRESS, TOKEN_BRIDGE_ABI, provider)

  console.log(`\n== On-chain state configuration ==\n`)

  const implementation = await tokenBridge.tokenImplementation()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`Implementation: \t\t${implementation}`)

  const chainId = await tokenBridge.chainId()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`chainId:\t\t\t${chainId}`)

  const governanceChainId = await tokenBridge.governanceChainId()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`governanceChainId:\t\t${governanceChainId}`)

  const finality = await tokenBridge.finality()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`finality:\t\t\t${finality}`)

  const governanceContract = await tokenBridge.governanceContract()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`governanceContract:\t\t${governanceContract}`)

  const weth = await tokenBridge.WETH()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`weth:\t\t\t\t${weth}`)

  const wormhole = await tokenBridge.wormhole()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`wormhole:\t\t\t${wormhole}`)

  const evmChainId = await tokenBridge.evmChainId()
  await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  console.log(`evmChainId:\t\t\t${evmChainId}`)
  
  console.log(`\n== Configuration checks ==\n`)

  if (implementation !== TOKEN_BRIDGE_IMPLEMENTATION) {
    console.log(`❌ tokenImplementation (${implementation}) does not match TOKEN_BRIDGE_IMPLEMENTATION (${TOKEN_BRIDGE_IMPLEMENTATION})`)
  } else {
    console.log(`✅ tokenImplementation (${implementation}) matches TOKEN_BRIDGE_IMPLEMENTATION (${TOKEN_BRIDGE_IMPLEMENTATION})`)
  }

  if (chainId !== BigInt(WORMHOLE_CHAIN_ID)) {
    console.log(`❌ chainId (${chainId}) does not match CHAIN_ID (${WORMHOLE_CHAIN_ID})`)
  } else {
    console.log(`✅ chainId (${chainId}) matches CHAIN_ID (${WORMHOLE_CHAIN_ID})`)
  }

  if (governanceChainId !== BigInt(GOVERNANCE_CHAIN_ID)) {
    console.log(`❌ governanceChainId (${governanceChainId}) does not match GOVERNANCE_CHAIN_ID (${GOVERNANCE_CHAIN_ID})`)
  } else {
    console.log(`✅ governanceChainId (${governanceChainId}) matches GOVERNANCE_CHAIN_ID (${GOVERNANCE_CHAIN_ID})`)
  }

  if (finality !== BigInt(FINALITY)) {
    console.log(`❌ finality (${finality}) does not match FINALITY (${FINALITY})`)
  } else {
    console.log(`✅ finality (${finality}) matches FINALITY (${FINALITY})`)
  }

  if (governanceContract !== GOVERNANCE_CONTRACT) {
    console.log(`❌ governanceContract (${governanceContract}) does not match GOVERNANCE_CONTRACT (${GOVERNANCE_CONTRACT})`)
  } else {
    console.log(`✅ governanceContract (${governanceContract}) matches GOVERNANCE_CONTRACT (${GOVERNANCE_CONTRACT})`)
  }

  if (weth !== WETH) {
    console.log(`❌ weth (${weth}) does not match WETH (${WETH})`)
  } else {
    console.log(`✅ weth (${weth}) matches WETH (${WETH})`)
  }

  if (wormhole !== CORE_BRIDGE_ADDRESS) {
    console.log(`❌ wormhole (${wormhole}) does not match CORE_BRIDGE_ADDRESS (${CORE_BRIDGE_ADDRESS})`)
  } else {
    console.log(`✅ wormhole (${wormhole}) matches CORE_BRIDGE_ADDRESS (${CORE_BRIDGE_ADDRESS})`)
  }

  if (evmChainId !== BigInt(EVM_CHAIN_ID)) {
    console.log(`❌ evmChainId (${evmChainId}) does not match EVM_CHAIN_ID (${EVM_CHAIN_ID})`)
  } else {
    console.log(`✅ evmChainId (${evmChainId}) matches EVM_CHAIN_ID (${EVM_CHAIN_ID})`)
  }

  console.log(`\n== TokenBridge chains registration ==\n`)

  for (const chainId of chainIds) {
    if(chainId == WORMHOLE_CHAIN_ID) {
      continue
    }
    const chain = chainIdToChain(chainId)
    const platformName = chainToPlatform(chain)
    const chainTokenBridge = tokenBridgeContracts("Mainnet", chain as any)
    if(!chainTokenBridge) {
      continue
    }
    const registeredTokenBridgeAddress = await tokenBridge.bridgeContracts(chainId)
    const universalAddress = toUniversal(chain, chainTokenBridge).toString()
    console.log(`[${chain} (${platformName})] chain token bridge: ${chainTokenBridge} | registered token bridge: ${registeredTokenBridgeAddress} | universal address: ${universalAddress}`)
    
    if(!checkTokenBridgeRegistration(chain, registeredTokenBridgeAddress, universalAddress)) {
      console.log(`❌ [${chain}] chain token bridge (${universalAddress}) does not match registered token bridge (${registeredTokenBridgeAddress})`)
    } else {
      console.log(`✅ [${chain}] chain token bridge (${universalAddress}) matches registered token bridge (${registeredTokenBridgeAddress})`)
    }
    await sleep(SLEEP_BETWEEN_RPC_CALLS_IN_MS)
  }
}

function checkTokenBridgeRegistration(chain: string, registered: string, universal: string) {
  if(chain === "Solana") {
    return registered === "0xec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5"
  }
  if(chain === "Aptos") {
    return registered === "0x0000000000000000000000000000000000000000000000000000000000000001"
  }
  if(chain === "Sui") {
    return registered === "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5"
  }
  return registered === universal
}

main()
