import { BridgeImplementation__factory, Implementation__factory, NFTBridgeImplementation__factory } from "@certusone/wormhole-sdk"
import { ethers } from "ethers"
import { NETWORKS } from "./networks"
import { impossible, Payload } from "./vaa"
import { Contracts, CONTRACTS, EVMChainName } from "../../sdk/js/src/utils/consts"

export async function execute_governance_evm(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET",
  chain: EVMChainName
) {
  let n = NETWORKS[network][chain]
  if (!n.rpc) {
    throw Error(`No ${network} rpc defined for ${chain} (see networks.ts)`)
  }
  if (!n.key) {
    throw Error(`No ${network} key defined for ${chain} (see networks.ts)`)
  }
  let rpc: string = n.rpc
  let key: string = n.key

  let contracts: Contracts = CONTRACTS[network][chain]

  let provider = new ethers.providers.JsonRpcProvider(rpc)
  let signer = new ethers.Wallet(key, provider)

  switch (payload.module) {
    case "Core":
      if (contracts.core === undefined) {
        throw Error(`Unknown core contract on ${network} for ${chain}`)
      }
      let c = new Implementation__factory(signer)
      let cb = c.attach(contracts.core)
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set")
          console.log("Hash: " + (await cb.submitNewGuardianSet(vaa)).hash)
          break
        case "ContractUpgrade":
          console.log("Upgrading core contract")
          console.log("Hash: " + (await cb.submitContractUpgrade(vaa)).hash)
          break
        default:
          impossible(payload)
      }
      break
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        throw Error(`Unknown nft bridge contract on ${network} for ${chain}`)
      }
      let n = new NFTBridgeImplementation__factory(signer)
      let nb = n.attach(contracts.nft_bridge)
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          console.log("Hash: " + (await nb.upgrade(vaa)).hash)
          console.log("Don't forget to verify the new implementation! See ethereum/VERIFY.md for instructions")
          break
        case "RegisterChain":
          console.log("Registering chain")
          console.log("Hash: " + (await nb.registerChain(vaa)).hash)
          break
        default:
          impossible(payload)

      }
      break
    case "TokenBridge":
      if (contracts.token_bridge === undefined) {
        throw Error(`Unknown token bridge contract on ${network} for ${chain}`)
      }
      let t = new BridgeImplementation__factory(signer)
      let tb = t.attach(contracts.token_bridge)
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          console.log("Hash: " + (await tb.upgrade(vaa)).hash)
          console.log("Don't forget to verify the new implementation! See ethereum/VERIFY.md for instructions")
          break
        case "RegisterChain":
          console.log("Registering chain")
          console.log("Hash: " + (await tb.registerChain(vaa)).hash)
          break
        default:
          impossible(payload)

      }
      break
    default:
      impossible(payload)
  }
}
