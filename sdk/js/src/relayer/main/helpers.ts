import {
  ChainId,
  CHAIN_ID_TO_NAME,
  CHAINS,
  isChain,
  CONTRACTS,
  getSignedVAAWithRetry,
  Network,
  parseVaa,
  ParsedVaa,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk"
import { GetSignedVAAResponse } from "@certusone/wormhole-sdk-proto-web/lib/cjs/publicrpc/v1/publicrpc"
import { Implementation__factory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts"
import { BigNumber, ContractReceipt, ethers, providers } from "ethers"
import {
  getWormholeRelayer,
  getWormholeRelayerAddress,
  RPCS_BY_CHAIN,
  GUARDIAN_RPC_HOSTS,
} from "../consts"
import {
  parseWormholeRelayerPayloadType,
  RelayerPayloadId,
  parseWormholeRelayerSend,
  parseWormholeRelayerResend,
  DeliveryInstruction,
  DeliveryInstructionsContainer,
  RedeliveryByTxHashInstruction,
  ExecutionParameters,
  MessageInfoType,
  DeliveryStatus
} from "../structs"
import { DeliveryEvent } from "../ethers-contracts/CoreRelayer"
type DeliveryTargetInfo = {
  status: DeliveryStatus | string
  deliveryTxHash: string | null
  vaaHash: string | null
  sourceChain: number | null
  sourceVaaSequence: BigNumber | null
}


export function parseWormholeLog(log: ethers.providers.Log): {
  type: RelayerPayloadId
  parsed: DeliveryInstructionsContainer | RedeliveryByTxHashInstruction | string
} {
  const abi = [
    "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);",
  ]
  const iface = new ethers.utils.Interface(abi)
  const parsed = iface.parseLog(log)
  const payload = Buffer.from(parsed.args.payload.substring(2), "hex")
  const type = parseWormholeRelayerPayloadType(payload)
  if (type == RelayerPayloadId.Delivery) {
    return { type, parsed: parseWormholeRelayerSend(payload) }
  } else if (type == RelayerPayloadId.Redelivery) {
    return { type, parsed: parseWormholeRelayerResend(payload) }
  } else {
    throw Error("Invalid wormhole log");
  }
}

export function printChain(chainId: number) {
  return `${CHAIN_ID_TO_NAME[chainId as ChainId]} (Chain ${chainId})`
}

export function getDefaultProvider(network: Network, chainId: ChainId) {
  return new ethers.providers.StaticJsonRpcProvider(
    RPCS_BY_CHAIN[network][CHAIN_ID_TO_NAME[chainId]]
  )
}


export function getBlockRange(provider: ethers.providers.Provider, timestamp?: number): [ethers.providers.BlockTag, ethers.providers.BlockTag] {
  return [-2040, "latest"]
}

export async function getWormholeRelayerDeliveryEventsBySourceSequence(
  environment: Network,
  targetChain: ChainId,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: number,
  sourceVaaSequence: BigNumber,
  blockStartNumber: ethers.providers.BlockTag,
  blockEndNumber: ethers.providers.BlockTag
): Promise<DeliveryTargetInfo[]> {
  const coreRelayer = getWormholeRelayer(targetChain, environment, targetChainProvider)

  const deliveryEvents = coreRelayer.filters.Delivery(
    null,
    sourceChain,
    sourceVaaSequence
  )

  // There is a max limit on RPCs sometimes for how many blocks to query
  return await transformDeliveryEvents(
    await coreRelayer.queryFilter(deliveryEvents, blockStartNumber, blockEndNumber),
    targetChainProvider
  )
}

export function deliveryStatus(status: number) {
  switch (status) {
    case 0:
      return DeliveryStatus.DeliverySuccess
    case 1:
      return DeliveryStatus.ReceiverFailure
    case 2:
      return DeliveryStatus.ForwardRequestFailure
    case 3:
      return DeliveryStatus.ForwardRequestSuccess
    case 4:
      return DeliveryStatus.InvalidRedelivery
    default:
      return DeliveryStatus.ThisShouldNeverHappen
  }
}

async function transformDeliveryEvents(
  events: DeliveryEvent[],
  targetProvider: ethers.providers.Provider
): Promise<DeliveryTargetInfo[]> {
  return Promise.all(
    events.map(async (x) => {
      return {
        status: deliveryStatus(x.args[4]),
        deliveryTxHash: x.transactionHash,
        vaaHash: x.args[3],
        sourceVaaSequence: x.args[2],
        sourceChain: x.args[1],
      }
    })
  )
}

export function getWormholeRelayerLog(
  receipt: ContractReceipt,
  bridgeAddress: string,
  emitterAddress: string,
  index: number,
): { log: ethers.providers.Log; sequence: string } {
  const bridgeLogs = receipt.logs.filter((l) => {
    return l.address === bridgeAddress
  })

  if (bridgeLogs.length == 0) {
    throw Error("No core contract interactions found for this transaction.")
  }

  const parsed = bridgeLogs.map((bridgeLog) => {
    const log = Implementation__factory.createInterface().parseLog(bridgeLog)
    return {
      sequence: log.args[1].toString(),
      nonce: log.args[2].toString(),
      emitterAddress: tryNativeToHexString(log.args[0].toString(), "ethereum"),
      log: bridgeLog,
    }
  })

  const filtered = parsed.filter(
    (x) =>
      x.emitterAddress == emitterAddress.toLowerCase() 
  )

  if (filtered.length == 0) {
    throw Error("No CoreRelayer contract interactions found for this transaction.")
  }

  if (index >= filtered.length) {
    throw Error("Specified delivery index is out of range.")
  } else {
    return {
      log: filtered[index].log,
      sequence: filtered[index].sequence,
    }
  }
}
