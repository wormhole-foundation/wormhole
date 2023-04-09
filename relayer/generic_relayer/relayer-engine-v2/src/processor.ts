import * as wh from "@certusone/wormhole-sdk"
import { Next } from "wormhole-relayer"
import {
  IDelivery,
  MessageInfoType,
  RelayerPayloadId,
  CoreRelayer__factory,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
} from "../pkgs/sdk/src"
import { EVMChainId } from "@certusone/wormhole-sdk"
import { GRContext } from "./app"

export async function processGenericRelayerVaa(ctx: GRContext, next: Next) {
  ctx.logger.info(`Processing generic relayer vaa`)
  const payloadId = parseWormholeRelayerPayloadType(ctx.vaa!.payload)
  // route payload types
  if (payloadId != RelayerPayloadId.Delivery) {
    ctx.logger.error(`Expected GR Delivery payload type, found ${payloadId}`)
    throw new Error("Expected GR Delivery payload type")
  }
  await processDelivery(ctx)
  await next()
}

async function processDelivery(ctx: GRContext) {
  const chainId = ctx.vaa!.emitterChain as wh.EVMChainId
  const payload = parseWormholeRelayerSend(ctx.vaa!.payload)

  if (
    payload.messages.findIndex((m) => m.payloadType !== MessageInfoType.EMITTER_SEQUENCE) != -1
  ) {
    throw new Error(`Only supports EmitterSequence MessageInfoType`)
  }
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    manifest: payload.messages,
  })
  const fetchedVaas = await ctx.fetchVaas({
    ids: payload.messages.map((m) => ({
      emitterAddress: m.emitterAddress!,
      emitterChain: chainId,
      sequence: m.sequence!.toBigInt(),
    })),
    txHash: ctx.sourceTxHash,
  })
  ctx.logger.debug(`Vaas fetched`)
  for (let i = 0; i < payload.instructions.length; i++) {
    const ix = payload.instructions[i]
    ctx.logger.debug(
      `Processing instruction ${i + 1} of ${payload.instructions.length}`,
      { instruction: ix }
    )
    // const chainId = assertEvmChainId(ix.targetChain)
    const chainId = ix.targetChain as EVMChainId
    const budget = ix.receiverValueTarget.add(ix.maximumRefundTarget)

    await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
      const coreRelayer = CoreRelayer__factory.connect(
        ctx.wormholeRelayers[chainId],
        wallet
      )

      const input: IDelivery.TargetDeliveryParametersStruct = {
        encodedVMs: fetchedVaas.map((v) => v.bytes),
        encodedDeliveryVAA: ctx.vaaBytes!,
        multisendIndex: i,
        relayerRefundAddress: wallet.address,
      }

      ctx.logger.debug("Sending 'deliver' tx...")
      await coreRelayer
        .deliver(input, { value: budget, gasLimit: 3000000 })
        .then((x) => x.wait())

      ctx.logger.info(
        `Relayed instruction ${i + 1} of ${
          payload.instructions.length
        } to chain ${chainId}`
      )
    })
  }
}
