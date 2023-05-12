import * as wh from "@certusone/wormhole-sdk";
import { Next, ParsedVaaWithBytes, sleep } from "relayer-engine";
import {
  VaaKeyType,
  RelayerPayloadId,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
  deliveryInstructionsPrintable,
  vaaKeyPrintable,
  parseWormholeRelayerResend,
  RedeliveryInstruction,
  DeliveryInstruction,
  packOverrides,
  DeliveryOverrideArgs,
} from "@certusone/wormhole-sdk/lib/cjs/relayer";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";
import { BigNumber, ethers } from "ethers";
import { CoreRelayer__factory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";
import { TargetDeliveryParametersStruct} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts/IWormholeRelayer.sol/IWormholeRelayerDelivery";


export async function processGenericRelayerVaa(ctx: GRContext, next: Next) {
  ctx.logger.info(`Processing generic relayer vaa`);
  const payloadId = parseWormholeRelayerPayloadType(ctx.vaa!.payload);
  // route payload types
  if (payloadId == RelayerPayloadId.Delivery) {
    ctx.logger.info(`Detected delivery VAA, processing delivery payload...`);
    await processDelivery(ctx);
  } else if (payloadId == RelayerPayloadId.Redelivery) {
    ctx.logger.info(
      `Detected redelivery VAA, processing redelivery payload...`
    );
    await processRedelivery(ctx);
  } else {
    ctx.logger.error(`Expected GR Delivery payload type, found ${payloadId}`);
    throw new Error("Expected GR Delivery payload type");
  }
  await next();
}

async function processDelivery(ctx: GRContext) {
  const deliveryVaa = parseWormholeRelayerSend(ctx.vaa!.payload);
  const sourceRelayProvider = ethers.utils.getAddress(wh.tryUint8ArrayToNative(deliveryVaa.sourceRelayProvider, "ethereum"));
  if (
    sourceRelayProvider !==
    ctx.relayProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceRelayProvider: deliveryVaa.sourceRelayProvider,
    });
    return;
  }
  processDeliveryInstruction(ctx, deliveryVaa, ctx.vaaBytes!);
}

async function processRedelivery(ctx: GRContext) {
  const redeliveryVaa = parseWormholeRelayerResend(ctx.vaa!.payload);
  const sourceRelayProvider = ethers.utils.getAddress(wh.tryUint8ArrayToNative(redeliveryVaa.sourceRelayProvider, "ethereum"));
  if (
    sourceRelayProvider !==
    ctx.relayProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceRelayProvider: redeliveryVaa.sourceRelayProvider,
    });
    return;
  }

  if (redeliveryVaa.key.infoType != VaaKeyType.EMITTER_SEQUENCE) {
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }

  ctx.logger.info(
    `Redelivery requested for the following VAA: `,
    vaaKeyPrintable(redeliveryVaa.key)
  );

  let originalVaa = await ctx.fetchVaa(
    redeliveryVaa.key.chainId as wh.ChainId,
    Buffer.from(redeliveryVaa.key.emitterAddress!),
    redeliveryVaa.key.sequence!.toBigInt()
  );

  ctx.logger.info("Retrieved original VAA!");
  const delivery = parseWormholeRelayerSend(originalVaa.payload);
  if (!isValidRedelivery(ctx, delivery, redeliveryVaa)) {
    ctx.logger.info("Exiting redelivery process");
    return;
  } else {
    ctx.logger.info("Redelivery is valid, proceeding with redelivery");
    processDeliveryInstruction(ctx, delivery, originalVaa.bytes, {
      newReceiverValueTarget: redeliveryVaa.newReceiverValueTarget,
      newMaximumRefundTarget: redeliveryVaa.newMaximumRefundTarget,
      gasLimit: redeliveryVaa.executionParameters.gasLimit,
      redeliveryHash: ctx.vaa!.hash,
    });
  }
}

function isValidRedelivery(
  ctx: GRContext,
  delivery: DeliveryInstruction,
  redelivery: RedeliveryInstruction
): boolean {
  //TODO check that the delivery & redelivery chains agree!
  if (delivery.targetChainId != redelivery.targetChainId) {
    ctx.logger.info(
      "Redelivery targetChain does not match original delivery targetChain"
    );
    ctx.logger.info(
      "Original targetChain: " +
        delivery.targetChainId +
        " Redelivery targetChain: " +
        redelivery.targetChainId
    );
    return false;
  }

  //TODO check that the sourceRelayerAddress is one of this relayer's addresses
  if (!redelivery.sourceRelayProvider) {
  }

  if (delivery.maximumRefundTarget.gt(redelivery.newMaximumRefundTarget)) {
    ctx.logger.info(
      "Redelivery maximumRefundTarget is less than original delivery maximumRefundTarget"
    );
    ctx.logger.info(
      "Original refund: " +
        delivery.maximumRefundTarget.toBigInt().toLocaleString() +
        " Redelivery: " +
        redelivery.newMaximumRefundTarget.toBigInt().toLocaleString()
    );
    return false;
  }
  if (delivery.receiverValueTarget.gt(redelivery.newReceiverValueTarget)) {
    ctx.logger.info(
      "Redelivery receiverValueTarget is less than original delivery receiverValueTarget"
    );
    ctx.logger.info(
      "Original refund: " +
        delivery.receiverValueTarget.toBigInt().toLocaleString(),
      +" Redelivery: " +
        redelivery.newReceiverValueTarget.toBigInt().toLocaleString()
    );
    return false;
  }
  if (
    delivery.executionParameters.gasLimit >
    redelivery.executionParameters.gasLimit
  ) {
    ctx.logger.info(
      "Redelivery gasLimit is less than original delivery gasLimit"
    );
    ctx.logger.info(
      "Original refund: " + delivery.executionParameters.gasLimit,
      " Redelivery: " + redelivery.executionParameters.gasLimit
    );
    return false;
  }

  return true;
}

async function processDeliveryInstruction(
  ctx: GRContext,
  delivery: DeliveryInstruction,
  deliveryVaa: Buffer | Uint8Array,
  overrides?: DeliveryOverrideArgs
) {
  //TODO this check is not quite correct
  if (
    delivery.vaaKeys.findIndex(
      (m) => m.infoType !== VaaKeyType.EMITTER_SEQUENCE
    ) != -1
  ) {
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    vaaKeys: delivery.vaaKeys.map(vaaKeyPrintable),
  });

  const vaaIds = delivery.vaaKeys.map((m) => ({
    emitterAddress: m.emitterAddress!,
    emitterChain: m.chainId! as wh.ChainId,
    sequence: m.sequence!.toBigInt(),
  }));

  let results = await ctx.fetchVaas({
    ids: vaaIds,
    // txHash: ctx.sourceTxHash,
  });

  ctx.logger.debug(`Processing delivery`, {
    deliveryVaa: deliveryInstructionsPrintable(delivery),
  });
  // const chainId = assertEvmChainId(ix.targetChain)
  const chainId = delivery.targetChainId as EVMChainId;
  const receiverValue = overrides?.newReceiverValueTarget
    ? overrides.newReceiverValueTarget
    : delivery.receiverValueTarget;
  const maxRefund = overrides?.newMaximumRefundTarget
    ? overrides.newMaximumRefundTarget
    : delivery.maximumRefundTarget;
  const budget = receiverValue.add(maxRefund);

  await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
    const coreRelayer = CoreRelayer__factory.connect(
      ctx.wormholeRelayers[chainId],
      wallet
    );

    const input: TargetDeliveryParametersStruct = {
      encodedVMs: results.map((v) => v.bytes),
      encodedDeliveryVAA: deliveryVaa,
      relayerRefundAddress: wallet.address,
      overrides: overrides ? packOverrides(overrides) : [],
    };

    console.log(input);

    const gasUnitsEstimate = await coreRelayer.estimateGas.deliver(input, {
      value: budget,
      gasLimit: 3000000,
    });
    const gasPrice = await coreRelayer.provider.getGasPrice();
    const estimatedTransactionFee = gasPrice.mul(gasUnitsEstimate);
    const estimatedTransactionFeeEther = ethers.utils.formatEther(
      estimatedTransactionFee
    );
    ctx.logger.info(
      `Estimated transaction cost (ether): ${estimatedTransactionFeeEther}`,
      {
        gasUnitsEstimate: gasUnitsEstimate.toString(),
        gasPrice: gasPrice.toString(),
        estimatedTransactionFee: estimatedTransactionFee.toString(),
        estimatedTransactionFeeEther,
        valueEther: ethers.utils.formatEther(budget),
      }
    );
    process.stdout.write("");
    await sleep(200);
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
