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
import { getAppConfig } from "./env";
import { ethers } from "ethers";

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

  const appConfig = getAppConfig();

  //TODO not anything even resembling this
  try {
    for (let i = 0; i < vaaIds.length; i++) {
      const { vaaBytes: signedVAA } = await wh.getSignedVAAWithRetry(
        appConfig.wormholeRpcs,
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
      //Provider is exploding, not sure why.
      // uncomment to independently test ethers provider
      // const rpc: any = getAppConfig().providers.chains;
      // const obj = rpc[ix.targetChain];
      // const url = obj.endpoints[0];
      // let provider = new ethers.providers.StaticJsonRpcProvider(url);

      // console.log("url", url);
      // const blockNumber = await provider.getBlockNumber();
      // console.log("blocknumber", blockNumber);

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
      const receipt = await coreRelayer
        .deliver(input, { value: budget, gasLimit: 3000000 })
        .then((x: any) => x.wait());

      const relayerContractLog = receipt.logs?.find((x: any) => {
        return x.address === ctx.wormholeRelayers[chainId];
      });
      if (relayerContractLog) {
        const parsedLog = coreRelayer.interface.parseLog(relayerContractLog!);
        const recipientAddress = parsedLog.args[0];
        const sourceChain = parsedLog.args[1];
        const sourceSequence = parsedLog.args[2];
        const vaaHash = parsedLog.args[3];
        const status = parsedLog.args[4];

        ctx.logger.info("resultant event!");
        ctx.logger.info("recipientAddress: " + recipientAddress);
        ctx.logger.info("sourceChain: " + sourceChain);
        ctx.logger.info("sourceSequence: " + sourceSequence.toString());
        ctx.logger.info("vaaHash: " + vaaHash);
        ctx.logger.info("status: " + status);
      }
      ctx.logger.info(
        `Relayed instruction ${i + 1} of ${
          payload.instructions.length
        } to chain ${chainId}, tx hash: ${receipt.transactionHash}`
      );

      ctx.logger.info("exiting processor");
    });
  }
}
