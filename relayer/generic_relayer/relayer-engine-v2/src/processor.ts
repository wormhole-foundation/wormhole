import * as wh from "@certusone/wormhole-sdk";
import { Next } from "relayer-engine";
import {
  IDelivery,
  VaaKeyType,
  RelayerPayloadId,
  CoreRelayer__factory,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
} from "../pkgs/sdk/src";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";

import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { GetARGsTypeFromFactory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts/commons";
import { loadAppConfig } from "./env";
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
  const payload = parseWormholeRelayerSend(ctx.vaa!.payload);

  //TODO this check is not quite correct
  if (
    payload.vaaKeys.findIndex(
      (m) => m.payloadType !== VaaKeyType.EMITTER_SEQUENCE
    ) != -1
  ) {
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    manifest: payload.vaaKeys,
  });

  const vaaIds = payload.vaaKeys.map((m) => ({
    emitterAddress: m.emitterAddress!,
    emitterChain: m.chainId!,
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
        ctx.opts.wormholeRpcs!,
        parseInt(vaaIds[i].emitterChain.toString()) as any,
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

    const ix = payload;
    ctx.logger.debug(
      `Processing instruction ${1} of ${1}`,
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
        const logArgs = {
          recipientAddress: parsedLog.args[0],
          sourceChain: parsedLog.args[1],
          sourceSequence: parsedLog.args[2],
          vaaHash: parsedLog.args[3],
          status: parsedLog.args[4],
        };
        ctx.logger.info("Parsed Delivery event", logArgs);
        switch (logArgs.status) {
          case 0:
            ctx.logger.info("Delivery Success");
            break;
          case 1:
            ctx.logger.info("Receiver Failure");
            break;
          case 2:
            ctx.logger.info("Forwarding Failure");
            break;
        }
      }
      ctx.logger.info(
        `Relayed instruction ${1} of ${
          1
        } to chain ${chainId}, tx hash: ${receipt.transactionHash}`
      );

      ctx.logger.info("exiting processor");
    });
  
}
