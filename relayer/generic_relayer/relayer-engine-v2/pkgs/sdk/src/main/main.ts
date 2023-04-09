import {
  ChainId,
  CHAIN_ID_TO_NAME,
  isChain,
  CONTRACTS,
  tryNativeToHexString,
  Network,
} from "@certusone/wormhole-sdk"
import { BigNumber, ethers } from "ethers"
import { getWormholeRelayerAddress } from "../consts"
import {
  RelayerPayloadId,
  DeliveryInstruction,
  DeliveryInstructionsContainer,
  MessageInfoType,
  DeliveryStatus,
} from "../structs"
import {
  getDefaultProvider,
  printChain,
  getWormholeRelayerLog,
  parseWormholeLog,
  getBlockRange,
  getWormholeRelayerDeliveryEventsBySourceSequence,
} from "./helpers"

type InfoRequest = {
  environment: Network
  sourceChain: ChainId
  sourceTransaction: string
  sourceChainProvider?: ethers.providers.Provider
  targetChainProviders?: Map<number, ethers.providers.Provider>
  targetChainBlockRanges?: Map<
    number,
    [ethers.providers.BlockTag, ethers.providers.BlockTag]
  >
  coreRelayerWhMessageIndex?: number
}

export type DeliveryInfo = {
  type: RelayerPayloadId.Delivery
  sourceChainId: ChainId
  sourceTransactionHash: string
  sourceDeliverySequenceNumber: number
  deliveryInstructionsContainer: DeliveryInstructionsContainer
  targetChainStatuses: {
    chainId: ChainId
    events: { status: DeliveryStatus | string; transactionHash: string | null }[]
  }[]
}

export function printWormholeRelayerInfo(info: DeliveryInfo) {
  console.log(stringifyWormholeRelayerInfo(info))
}
export function stringifyWormholeRelayerInfo(info: DeliveryInfo): string {
  let stringifiedInfo = ""
  if (info.type == RelayerPayloadId.Delivery) {
    stringifiedInfo += `Found delivery request in transaction ${
      info.sourceTransactionHash
    } on ${printChain(info.sourceChainId)}\n`

    const numMsgs = info.deliveryInstructionsContainer.messages.length
    stringifiedInfo += `\nThe following ${numMsgs} messages were requested to be relayed:\n`
    stringifiedInfo += info.deliveryInstructionsContainer.messages.map((msgInfo, i) => {
      let result = ""
      result += `\n(Message ${i}): `
      if (msgInfo.payloadType == MessageInfoType.EMITTER_SEQUENCE) {
        result += `Message with emitter address ${msgInfo.emitterAddress?.toString(
          "hex"
        )} and sequence number ${msgInfo.sequence}\n`
      } else if (msgInfo.payloadType == MessageInfoType.VAAHASH) {
        result += `Message with VAA Hash ${msgInfo.vaaHash?.toString("hex")}\n`
      } else {
        result += `Message not specified correctly\n`
      }
    })

    const length = info.deliveryInstructionsContainer.instructions.length
    stringifiedInfo += `\nMessages were requested to be sent to ${length} destination${
      length == 1 ? "" : "s"
    }:\n`
    stringifiedInfo +=
      info.deliveryInstructionsContainer.instructions
        .map((instruction: DeliveryInstruction, i) => {
          let result = ""
          const targetChainName = CHAIN_ID_TO_NAME[instruction.targetChain as ChainId]
          result += `\n(Destination ${i}): Target address is 0x${instruction.targetAddress.toString(
            "hex"
          )} on ${printChain(instruction.targetChain)}\n`
          result += `Max amount to use for gas: ${instruction.maximumRefundTarget} of ${targetChainName} currency\n`
          result += instruction.receiverValueTarget.gt(0)
            ? `Amount to pass into target address: ${
                instruction.receiverValueTarget
              } of ${CHAIN_ID_TO_NAME[instruction.targetChain as ChainId]} currency\n`
            : ``
          result += `Gas limit: ${instruction.executionParameters.gasLimit} ${targetChainName} gas\n`
          result += `Relay Provider Delivery Address: 0x${instruction.executionParameters.providerDeliveryAddress.toString(
            "hex"
          )}\n`
          result += info.targetChainStatuses[i].events
            .map(
              (e, i) =>
                `Delivery attempt ${i + 1}: ${e.status}${
                  e.transactionHash
                    ? ` (${targetChainName} transaction hash: ${e.transactionHash})`
                    : ""
                }`
            )
            .join("\n")
          return result
        })
        .join("\n") + "\n"
  }
  return stringifiedInfo
}

export async function getWormholeRelayerInfo(
  infoRequest: InfoRequest
): Promise<DeliveryInfo > {
  const sourceChainProvider =
    infoRequest.sourceChainProvider ||
    getDefaultProvider(infoRequest.environment, infoRequest.sourceChain)
  if (!sourceChainProvider)
    throw Error(
      "No default RPC for this chain; pass in your own provider (as sourceChainProvider)"
    )
  const receipt = await sourceChainProvider.getTransactionReceipt(
    infoRequest.sourceTransaction
  )
  if (!receipt) throw Error("Transaction has not been mined")
  const bridgeAddress =
    CONTRACTS[infoRequest.environment][CHAIN_ID_TO_NAME[infoRequest.sourceChain]].core
  const coreRelayerAddress = getWormholeRelayerAddress(
    infoRequest.sourceChain,
    infoRequest.environment
  )
  if (!bridgeAddress || !coreRelayerAddress) {
    throw Error(
      `Invalid chain ID or network: Chain ID ${infoRequest.sourceChain}, ${infoRequest.environment}`
    )
  }

  const deliveryLog = getWormholeRelayerLog(
    receipt,
    bridgeAddress,
    tryNativeToHexString(coreRelayerAddress, "ethereum"),
    infoRequest.coreRelayerWhMessageIndex ? infoRequest.coreRelayerWhMessageIndex : 0
  )

  const { type, parsed } = parseWormholeLog(deliveryLog.log)

  const deliveryInstructionsContainer = parsed as DeliveryInstructionsContainer

  const targetChainStatuses = await Promise.all(
    deliveryInstructionsContainer.instructions.map(
      async (instruction: DeliveryInstruction) => {
        const targetChain = instruction.targetChain as ChainId
        if (!isChain(targetChain)) throw Error(`Invalid Chain: ${targetChain}`)
        const targetChainProvider =
          infoRequest.targetChainProviders?.get(targetChain) ||
          getDefaultProvider(infoRequest.environment, targetChain)

        if (!targetChainProvider)
          throw Error(
            "No default RPC for this chain; pass in your own provider (as targetChainProvider)"
          )

        const sourceChainBlock = await sourceChainProvider.getBlock(receipt.blockNumber)
        const [blockStartNumber, blockEndNumber] =
          infoRequest.targetChainBlockRanges?.get(targetChain) ||
          getBlockRange(targetChainProvider, sourceChainBlock.timestamp)

        const deliveryEvents = await getWormholeRelayerDeliveryEventsBySourceSequence(
          infoRequest.environment,
          targetChain,
          targetChainProvider,
          infoRequest.sourceChain,
          BigNumber.from(deliveryLog.sequence),
          blockStartNumber,
          blockEndNumber
        )
        if (deliveryEvents.length == 0) {
          let status = `Delivery didn't happen on ${printChain(
            targetChain
          )} within blocks ${blockStartNumber} to ${blockEndNumber}.`
          try {
            const blockStart = await targetChainProvider.getBlock(blockStartNumber)
            const blockEnd = await targetChainProvider.getBlock(blockEndNumber)
            status = `Delivery didn't happen on ${printChain(
              targetChain
            )} within blocks ${blockStart.number} to ${
              blockEnd.number
            } (within times ${new Date(
              blockStart.timestamp * 1000
            ).toString()} to ${new Date(blockEnd.timestamp * 1000).toString()})`
          } catch (e) {}
          deliveryEvents.push({
            status,
            deliveryTxHash: null,
            vaaHash: null,
            sourceChain: infoRequest.sourceChain,
            sourceVaaSequence: BigNumber.from(deliveryLog.sequence),
          })
        }
        return {
          chainId: targetChain,
          events: deliveryEvents.map((e) => ({
            status: e.status,
            transactionHash: e.deliveryTxHash,
          })),
        }
      }
    )
  )

  return {
    type,
    sourceChainId: infoRequest.sourceChain,
    sourceTransactionHash: infoRequest.sourceTransaction,
    sourceDeliverySequenceNumber: BigNumber.from(deliveryLog.sequence).toNumber(),
    deliveryInstructionsContainer,
    targetChainStatuses,
  }
}
