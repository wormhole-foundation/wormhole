import * as wh from "@certusone/wormhole-sdk";
import { Next } from "wormhole-relayer";
import {
  IDelivery,
  MessageInfoType,
  RelayerPayloadId,
  CoreRelayer__factory,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
} from "../pkgs/sdk/src";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";

import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { GetARGsTypeFromFactory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts/commons";

export async function processGenericRelayerVaa(ctx: GRContext, next: Next) {
  ctx.logger.info(`Processing generic relayer vaa`);
  const payloadId = parseWormholeRelayerPayloadType(ctx.vaa!.payload);
  // route payload types
  if (payloadId != RelayerPayloadId.Delivery) {
    ctx.logger.error(`Expected GR Delivery payload type, found ${payloadId}`);
    throw new Error("Expected GR Delivery payload type");
  }
  await processDelivery(ctx);
  await next();
}

async function processDelivery(ctx: GRContext) {
  const chainId = ctx.vaa!.emitterChain as wh.EVMChainId;
  const payload = parseWormholeRelayerSend(ctx.vaa!.payload);

  //TODO this check is not quite correct
  if (
    payload.messages.findIndex(
      (m) => m.payloadType !== MessageInfoType.EMITTER_SEQUENCE
    ) != -1
  ) {
    throw new Error(`Only supports EmitterSequence MessageInfoType`);
  }
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    manifest: payload.messages,
  });

  const vaaIds = payload.messages.map((m) => ({
    emitterAddress: m.emitterAddress!,
    emitterChain: chainId,
    sequence: m.sequence!.toBigInt(),
  }));

  ctx.logger.debug("about to fetch the following vaaIds: " + vaaIds);

  //Doesn't seem to work at the moment
  // const fetchedVaas = await ctx.fetchVaas({
  //   ids: vaaIds,
  //   txHash: ctx.sourceTxHash,
  // });

  const results: Uint8Array[] = [];

  //TODO not anything even resembling this
  try {
    for (let i = 0; i < vaaIds.length; i++) {
      const { vaaBytes: signedVAA } = await wh.getSignedVAAWithRetry(
        ["localhost:7071"],
        parseInt(chainId.toString()) as any,
        vaaIds[i].emitterAddress.toString("hex"),
        vaaIds[i].sequence.toString(),
        { transport: NodeHttpTransport() },
        1000,
        3
      );
      console.log("returned from the first signed VAA with retry");
      results.push(signedVAA);
    }
  } catch (e) {
    console.log(e);
    console.log("getSignedVAAWithRetry error");
  }

  ctx.logger.debug(`Vaas fetched`);
  for (let i = 0; i < payload.instructions.length; i++) {
    const ix = payload.instructions[i];
    ctx.logger.debug(
      `Processing instruction ${i + 1} of ${payload.instructions.length}`,
      { instruction: ix }
    );
    // const chainId = assertEvmChainId(ix.targetChain)
    const chainId = ix.targetChain as EVMChainId;
    const budget = ix.receiverValueTarget.add(ix.maximumRefundTarget);

    await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
      const coreRelayer = CoreRelayer__factory.connect(
        ctx.wormholeRelayers[chainId],
        wallet
      );

      const input: IDelivery.TargetDeliveryParametersStruct = {
        encodedVMs: results.map((v) => v),
        encodedDeliveryVAA: ctx.vaaBytes!,
        multisendIndex: i,
        relayerRefundAddress: wallet.address,
      };

      ctx.logger.debug("Sending 'deliver' tx...");
      await coreRelayer
        .deliver(input, { value: budget, gasLimit: 3000000 })
        .then((x) => x.wait());

      ctx.logger.info(
        `Relayed instruction ${i + 1} of ${
          payload.instructions.length
        } to chain ${chainId}`
      );
    });
  }
}
