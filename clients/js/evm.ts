import { ethers } from "ethers"
import { NETWORKS } from "./networks"
import { encode, Encoding, impossible, Payload, typeWidth } from "./vaa"
import axios from "axios";
import * as celo from "@celo-tools/celo-ethers-wrapper";
import { solidityKeccak256 } from "ethers/lib/utils"
import { CHAINS, CONTRACTS, Contracts, EVMChainName } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { BridgeImplementation__factory, Implementation__factory, NFTBridgeImplementation__factory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";

const _IMPLEMENTATION_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"

export async function query_contract_evm(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: EVMChainName,
  module: "Core" | "NFTBridge" | "TokenBridge",
  contract_address: string | undefined,
  _rpc: string | undefined
): Promise<object> {
  let n = NETWORKS[network][chain]
  let rpc: string | undefined = _rpc ?? n.rpc;
  if (rpc === undefined) {
    throw Error(`No ${network} rpc defined for ${chain} (see networks.ts)`)
  }

  let contracts: Contracts = CONTRACTS[network][chain]

  const provider = new ethers.providers.JsonRpcProvider(rpc)

  let result: any = {}

  switch (module) {
    case "Core":
      contract_address = contract_address ? contract_address : contracts.core;
      if (contract_address === undefined) {
        throw Error(`Unknown core contract on ${network} for ${chain}`)
      }
      const core = Implementation__factory.connect(contract_address, provider)
      result.address = contract_address
      result.currentGuardianSetIndex = await core.getCurrentGuardianSetIndex()
      result.guardianSet = {}
      for (let i of Array(result.currentGuardianSetIndex + 1).keys()) {
        let guardian_set = await core.getGuardianSet(i)
        result.guardianSet[i] = { keys: guardian_set[0], expiry: guardian_set[1] }
      }
      result.guardianSetExpiry = await core.getGuardianSetExpiry()
      result.chainId = await core.chainId()
      result.evmChainId = await core.evmChainId()
      result.isFork = await core.isFork()
      result.governanceChainId = await core.governanceChainId()
      result.governanceContract = await core.governanceContract()
      result.messageFee = await core.messageFee()
      result.implementation = (await getStorageAt(rpc, contract_address, _IMPLEMENTATION_SLOT, ["address"]))[0]
      result.isInitialized = await core.isInitialized(result.implementation)
      break
    case "TokenBridge":
      contract_address = contract_address ? contract_address : contracts.token_bridge;
      if (contract_address === undefined) {
        throw Error(`Unknown token bridge contract on ${network} for ${chain}`)
      }
      const tb = BridgeImplementation__factory.connect(contract_address, provider)
      result.address = contract_address
      result.wormhole = await tb.wormhole()
      result.implementation = (await getStorageAt(rpc, contract_address, _IMPLEMENTATION_SLOT, ["address"]))[0]
      result.isInitialized = await tb.isInitialized(result.implementation)
      result.tokenImplementation = await tb.tokenImplementation()
      result.chainId = await tb.chainId()
      result.finality = await tb.finality()
      result.evmChainId = (await tb.evmChainId()).toString()
      result.isFork = await tb.isFork()
      result.governanceChainId = await tb.governanceChainId()
      result.governanceContract = await tb.governanceContract()
      result.WETH = await tb.WETH()
      result.registrations = {}
      for (let [c_name, c_id] of Object.entries(CHAINS)) {
        if (c_name === chain || c_name === "unset") {
          continue
        }
        result.registrations[c_name] = await tb.bridgeContracts(c_id)
      }
      break
    case "NFTBridge":
      contract_address = contract_address ? contract_address : contracts.nft_bridge;
      if (contract_address === undefined) {
        throw Error(`Unknown nft bridge contract on ${network} for ${chain}`)
      }
      const nb = NFTBridgeImplementation__factory.connect(contract_address, provider)
      result.address = contract_address
      result.wormhole = await nb.wormhole()
      result.implementation = (await getStorageAt(rpc, contract_address, _IMPLEMENTATION_SLOT, ["address"]))[0]
      result.isInitialized = await nb.isInitialized(result.implementation)
      result.tokenImplementation = await nb.tokenImplementation()
      result.chainId = await nb.chainId()
      result.finality = await nb.finality()
      result.evmChainId = (await nb.evmChainId()).toString()
      result.isFork = await nb.isFork()
      result.governanceChainId = await nb.governanceChainId()
      result.governanceContract = await nb.governanceContract()
      result.registrations = {}
      for (let [c_name, c_id] of Object.entries(CHAINS)) {
        if (c_name === chain || c_name === "unset") {
          continue
        }
        result.registrations[c_name] = await nb.bridgeContracts(c_id)
      }
      break
    default:
      impossible(module)
  }

  return result
}

export async function getImplementation(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: EVMChainName,
  module: "Core" | "NFTBridge" | "TokenBridge",
  contract_address: string | undefined,
  _rpc: string | undefined
): Promise<ethers.BigNumber> {
  let n = NETWORKS[network][chain]
  let rpc: string | undefined = _rpc ?? n.rpc;
  if (rpc === undefined) {
    throw Error(`No ${network} rpc defined for ${chain} (see networks.ts)`)
  }

  let contracts: Contracts = CONTRACTS[network][chain]

  switch (module) {
    case "Core":
      contract_address = contract_address ? contract_address : contracts.core;
      break
    case "TokenBridge":
      contract_address = contract_address ? contract_address : contracts.token_bridge;
      break
    case "NFTBridge":
      contract_address = contract_address ? contract_address : contracts.nft_bridge;
      break
    default:
      impossible(module)
  }

  return (await getStorageAt(rpc, contract_address, _IMPLEMENTATION_SLOT, ["address"]))[0]
}

export async function execute_evm(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: EVMChainName,
  contract_address: string | undefined,
  _rpc: string | undefined
) {
  let n = NETWORKS[network][chain]
  let rpc: string | undefined = _rpc ?? n.rpc;
  if (rpc === undefined) {
    throw Error(`No ${network} rpc defined for ${chain} (see networks.ts)`)
  }
  if (!n.key) {
    throw Error(`No ${network} key defined for ${chain} (see networks.ts)`)
  }
  let key: string = n.key

  let contracts: Contracts = CONTRACTS[network][chain]

  let provider: ethers.providers.JsonRpcProvider;
  let signer: ethers.Wallet;
  if (chain === "celo") {
    provider = new celo.CeloProvider(rpc)
    await provider.ready
    signer = new celo.CeloWallet(key, provider)
  } else {
    provider = new ethers.providers.JsonRpcProvider(rpc)
    signer = new ethers.Wallet(key, provider)
  }

  // Here we apply a set of chain-specific overrides.
  // NOTE: some of these might have only been tested on mainnet. If it fails in
  // testnet (or devnet), they might require additional guards
  let overrides: ethers.Overrides = {}
  if (chain === "karura" || chain == "acala") {
    overrides = await getKaruraGasParams(n.rpc)
  } else if (chain === "polygon") {
    let feeData = await provider.getFeeData();
    overrides = {
      maxFeePerGas: feeData.maxFeePerGas?.mul(50) || undefined,
      maxPriorityFeePerGas: feeData.maxPriorityFeePerGas?.mul(50) || undefined,
    };
  } else if (chain === "klaytn" || chain === "fantom") {
    overrides = { gasPrice: (await signer.getGasPrice()).toString() }
  }

  switch (payload.module) {
    case "Core":
      contract_address = contract_address ? contract_address : contracts.core;
      if (contract_address === undefined) {
        throw Error(`Unknown core contract on ${network} for ${chain}`)
      }
      let c = new Implementation__factory(signer)
      let cb = c.attach(contract_address)
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set")
          console.log("Hash: " + (await cb.submitNewGuardianSet(vaa, overrides)).hash)
          break
        case "ContractUpgrade":
          console.log("Upgrading core contract")
          console.log("Hash: " + (await cb.submitContractUpgrade(vaa, overrides)).hash)
          break
        default:
          impossible(payload)
      }
      break
    case "NFTBridge":
      contract_address = contract_address ? contract_address : contracts.nft_bridge;
      if (contract_address === undefined) {
        throw Error(`Unknown nft bridge contract on ${network} for ${chain}`)
      }
      let n = new NFTBridgeImplementation__factory(signer)
      let nb = n.attach(contract_address)
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          console.log("Hash: " + (await nb.upgrade(vaa, overrides)).hash)
          console.log("Don't forget to verify the new implementation! See ethereum/VERIFY.md for instructions")
          break
        case "RegisterChain":
          console.log("Registering chain")
          console.log("Hash: " + (await nb.registerChain(vaa, overrides)).hash)
          break
        case "Transfer":
          console.log("Completing transfer")
          console.log("Hash: " + (await nb.completeTransfer(vaa, overrides)).hash)
          break
        default:
          impossible(payload)

      }
      break
    case "TokenBridge":
      contract_address = contract_address ? contract_address : contracts.token_bridge;
      if (contract_address === undefined) {
        throw Error(`Unknown token bridge contract on ${network} for ${chain}`)
      }
      let t = new BridgeImplementation__factory(signer)
      let tb = t.attach(contract_address)
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          console.log("Hash: " + (await tb.upgrade(vaa, overrides)).hash)
          console.log("Don't forget to verify the new implementation! See ethereum/VERIFY.md for instructions")
          break
        case "RegisterChain":
          console.log("Registering chain")
          console.log("Hash: " + (await tb.registerChain(vaa, overrides)).hash)
          break
        case "Transfer":
          console.log("Completing transfer")
          console.log("Hash: " + (await tb.completeTransfer(vaa, overrides)).hash)
          break
        case "AttestMeta":
          console.log("Creating wrapped token")
          console.log("Hash: " + (await tb.createWrapped(vaa, overrides)).hash)
          break
        case "TransferWithPayload":
          console.log("Completing transfer with payload")
          console.log("Hash: " + (await tb.completeTransferWithPayload(vaa, overrides)).hash)
          break
        default:
          impossible(payload)
          break

      }
      break
    default:
      impossible(payload)
  }
}

/**
 *
 * Hijack a core contract. This function is useful when working with a mainnet
 * fork (hardhat or anvil). A fork of the mainnet contract will naturally store
 * the mainnet guardian set, so we can't readily interact with these contracts,
 * because we can't forge signed VAAs for those guardians. This function uses
 * [[setStorageAt]] to override the guardian set to something we have the
 * private keys for (typically the devnet guardian used for testing).
 * This way we can test contract upgrades before rolling them out on mainnet.
 *
 * @param rpc the JSON RPC endpoint (needs to be hardhat of anvil)
 * @param contract_address address of the core bridge contract
 * @param guardian_addresses addresses of the desired guardian set to upgrade to
 * @param new_guardian_set_index if specified, the new guardian set will be
 * written into this guardian set index, and the guardian set index of the
 * contract changed to it.
 * If unspecified, then the current guardian set index will be overridden.
 * In particular, it's possible to both upgrade or downgrade the guardian set
 * this way. The latter is useful for testing locally if you already have some
 * VAAs handy that are signed by guardian set 0.
 */
export async function hijack_evm(
  rpc: string,
  contract_address: string,
  guardian_addresses: string[],
  new_guardian_set_index: number | undefined
): Promise<void> {
  const GUARDIAN_SETS_SLOT = 0x02
  const GUARDIAN_SET_INDEX_SLOT = 0x3

  const provider = new ethers.providers.JsonRpcProvider(rpc)
  const core = Implementation__factory.connect(contract_address, provider)
  let guardianSetIndex: number
  let guardianSetExpiry: number
  [guardianSetIndex, guardianSetExpiry] = await getStorageAt(rpc, contract_address, GUARDIAN_SET_INDEX_SLOT, ["uint32", "uint32"])
  console.log("Attempting to hijack core bridge guardian set.")
  const current_set = await core.getGuardianSet(guardianSetIndex)
  console.log(`Current guardian set (index ${guardianSetIndex}):`)
  console.log(current_set[0])

  if (new_guardian_set_index !== undefined) {
    await setStorageAt(rpc, contract_address, GUARDIAN_SET_INDEX_SLOT, ["uint32", "uint32"], [new_guardian_set_index, guardianSetExpiry])
    guardianSetIndex = await core.getCurrentGuardianSetIndex()
    if (new_guardian_set_index !== guardianSetIndex) {
      throw Error("Failed to update guardian set index.")
    } else {
      console.log(`Guardian set index updated to ${new_guardian_set_index}`)
    }
  }
  const addresses_slot = computeMappingElemSlot(GUARDIAN_SETS_SLOT, guardianSetIndex)
  console.log(`Writing new set of guardians into set ${guardianSetIndex}...`)
  guardian_addresses.forEach(async (address, i) => {
    await setStorageAt(rpc, contract_address, computeArrayElemSlot(addresses_slot, i), ["address"], [address])
  })
  await setStorageAt(rpc, contract_address, addresses_slot, ["uint256"], [guardian_addresses.length])
  const after_guardian_set_index = await core.getCurrentGuardianSetIndex()
  const new_set = await core.getGuardianSet(after_guardian_set_index)
  console.log(`Current guardian set (index ${after_guardian_set_index}):`)
  console.log(new_set[0])
  console.log("Success.")
}

async function getKaruraGasParams(rpc: string): Promise<{
  gasPrice: number;
  gasLimit: number;
}> {
  const gasLimit = 21000000;
  const storageLimit = 64001;
  const res = (
    await axios.post(rpc, {
      id: 0,
      jsonrpc: "2.0",
      method: "eth_getEthGas",
      params: [
        {
          gasLimit,
          storageLimit,
        },
      ],
    })
  ).data.result;

  return {
    gasLimit: parseInt(res.gasLimit, 16),
    gasPrice: parseInt(res.gasPrice, 16),
  };
}

////////////////////////////////////////////////////////////////////////////////
// Storage manipulation
//
// Below we define a set of utilities for working with the EVM storage. For
// reference on storage layout, see [1].
//
// [1]: https://docs.soliditylang.org/en/v0.8.14/internals/layout_in_storage.html

export type StorageSlot = ethers.BigNumber
// we're a little more permissive in contravariant positions...
export type StorageSlotish = ethers.BigNumberish

/**
 *
 * Compute the storage slot of an array element.
 *
 * @param array_slot the storage slot of the array variable
 * @param offset the index of the element to compute the storage slot for
 */
export function computeArrayElemSlot(array_slot: StorageSlotish, offset: number): StorageSlot {
  return ethers.BigNumber.from(solidityKeccak256(["bytes"], [array_slot])).add(offset)
}

/**
 *
 * Compute the storage slot of a mapping key.
 *
 * @param map_slot the storage slot of the mapping variable
 * @param key the key to compute the storage slot for
 */
export function computeMappingElemSlot(map_slot: StorageSlotish, key: any): StorageSlot {
  const slot_preimage = ethers.utils.defaultAbiCoder.encode(["uint256", "uint256"], [key, map_slot])
  return ethers.BigNumber.from(solidityKeccak256(["bytes"], [slot_preimage]))
}

/**
 *
 * Get the values stored in a storage slot. [[ethers.Provider.getStorageAt]]
 * returns the whole slot as one 32 byte value, but if there are multiple values
 * stored in the slot (which solidity does to save gas), it is useful to parse
 * the output accordingly. This function is a wrapper around the storage query
 * provided by [[ethers]] that does the additional parsing.
 *
 * @param rpc the JSON RPC endpoint
 * @param contract_address address of the contract to be queried
 * @param storage_slot the storage slot to query
 * @param types The types of values stored in the storage slot. It's a list,
 * because solidity packs multiple values into a single storage slot to save gas
 * when the elements fit.
 *
 * @returns _values the values to write into the slot (packed)
 */
async function getStorageAt(rpc: string, contract_address: string, storage_slot: StorageSlotish, types: Encoding[]): Promise<any[]> {
  const total = types.map((typ) => typeWidth(typ)).reduce((x, y) => (x + y))
  if (total > 32) {
    throw new Error(`Storage slots can contain a maximum of 32 bytes. Total size of ${types} is ${total} bytes.`)
  }

  const string_val: string =
    await (new ethers.providers.JsonRpcProvider(rpc).getStorageAt(contract_address, storage_slot))
  let val = ethers.BigNumber.from(string_val)
  let ret: any[] = []
  // we decode the elements one by one, by shifting down the stuff we've parsed already
  types.forEach((typ) => {
    const padded = ethers.utils.defaultAbiCoder.encode(["uint256"], [val])
    ret.push(ethers.utils.defaultAbiCoder.decode([typ], padded)[0])
    val = val.shr(typeWidth(typ) * 8)
  })
  return ret
}

/**
 *
 * Use the 'hardhat_setStorageAt' rpc method to override a storage slot of a
 * contract. This method is understood by both hardhat and anvil (from foundry).
 * Useful for manipulating the storage of a forked mainnet contract (such as for
 * changing the guardian set to allow submitting VAAs to).
 *
 * @param rpc the JSON RPC endpoint (needs to be hardhat of anvil)
 * @param contract_address address of the contract to be queried
 * @param storage_slot the storage slot to query
 * @param types The types of values stored in the storage slot. It's a list,
 * because solidity packs multiple values into a single storage slot to save gas
 * when the elements fit. This means that when writing into the slot, all values
 * must be accounted for, otherwise we end up zeroing out some fields.
 * @param values the values to write into the slot (packed)
 *
 * @returns the `data` property of the JSON response
 */
export async function setStorageAt(rpc: string, contract_address: string, storage_slot: StorageSlotish, types: Encoding[], values: any[]): Promise<any> {
  // we need to reverse the values and types arrays, because the first element
  // is stored at the rightmost bytes.
  //
  // for example:
  //   uint32 a
  //   uint32 b
  // will be stored as 0x...b...a
  const _values = values.reverse()
  const _types = types.reverse()
  const total = _types.map((typ) => typeWidth(typ)).reduce((x, y) => (x + y))
  // ensure that the types fit into a slot
  if (total > 32) {
    throw new Error(`Storage slots can contain a maximum of 32 bytes. Total size of ${_types} is ${total} bytes.`)
  }
  if (_types.length !== _values.length) {
    throw new Error(`Expected ${_types.length} value(s), but got ${_values.length}.`)
  }
  // as far as I could tell, `ethers` doesn't provide a way to pack multiple
  // values into a single slot (the abi coder pads everything to 32 bytes), so we do it ourselves
  const val = "0x" + _types.map((typ, i) => encode(typ, _values[i])).reduce((x, y) => x + y).padStart(64, "0")
  // format the storage slot
  const slot = ethers.utils.defaultAbiCoder.encode(["uint256"], [storage_slot])
  console.log(`slot ${slot} := ${val}`)

  return (await axios.post(rpc, {
    id: 0,
    jsonrpc: "2.0",
    method: "hardhat_setStorageAt",
    params: [
      contract_address,
      slot,
      val,
    ],
  })).data
}
