import * as wh from "@certusone/wormhole-sdk";
import { Next } from "relayer-engine";
import {
  IDelivery,
  VaaKeyType,
  RelayerPayloadId,
  CoreRelayer__factory,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
  deliveryInstructionsPrintable,
  vaaKeyPrintable,
} from "../pkgs/sdk/src";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";
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
  const deliveryVaa = parseWormholeRelayerSend(ctx.vaa!.payload);

  //TODO this check is not quite correct
  if (
    deliveryVaa.vaaKeys.findIndex(
      (m) => m.payloadType !== VaaKeyType.EMITTER_SEQUENCE
    ) != -1
  ) {
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    vaaKeys: deliveryVaa.vaaKeys.map(vaaKeyPrintable),
  });

  const vaaIds = deliveryVaa.vaaKeys.map((m) => ({
    emitterAddress: m.emitterAddress!,
    emitterChain: m.chainId! as wh.ChainId,
    sequence: m.sequence!.toBigInt(),
  }));

  let results = await ctx.fetchVaas({
    ids: vaaIds,
    // txHash: ctx.sourceTxHash,
  });

  ctx.logger.debug(`Processing delivery`, {
    deliveryVaa: deliveryInstructionsPrintable(deliveryVaa),
  });
  // const chainId = assertEvmChainId(ix.targetChain)
  const chainId = deliveryVaa.targetChain as EVMChainId;
  const budget = deliveryVaa.receiverValueTarget.add(
    deliveryVaa.maximumRefundTarget
  );

  await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
    const coreRelayer = CoreRelayer__factory.connect(
      ctx.wormholeRelayers[chainId],
      wallet
    );

    const input: IDelivery.TargetDeliveryParametersStruct = {
      encodedVMs: results.map((v) => v.bytes),
      encodedDeliveryVAA: ctx.vaaBytes!,
      relayerRefundAddress: wallet.address,
    };

    ctx.logger.debug("Sending 'deliver' tx...");
    const receipt = await coreRelayer
      .deliver(input, { value: budget, gasLimit: 3000000 })
      .then((x: any) => x.wait());

    logResults(ctx, receipt, chainId);
  });
}

function logResults(
  ctx: GRContext,
  receipt: ethers.ContractReceipt,
  chainId: EVMChainId
) {
  const relayerContractLog = receipt.logs?.find((x: any) => {
    return x.address === ctx.wormholeRelayers[chainId];
  });
  if (relayerContractLog) {
    const parsedLog = CoreRelayer__factory.createInterface().parseLog(
      relayerContractLog!
    );
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
    `Relayed instruction to chain ${chainId}, tx hash: ${receipt.transactionHash}`
  );
}
