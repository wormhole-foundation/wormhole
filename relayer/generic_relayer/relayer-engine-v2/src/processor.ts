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
  parseEVMExecutionInfoV1
} from "@certusone/wormhole-sdk/lib/cjs/relayer";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";
import { BigNumber, ethers } from "ethers";
import { IWormholeRelayerDelivery__factory } from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";


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
  const sourceDeliveryProvider = ethers.utils.getAddress(wh.tryUint8ArrayToNative(deliveryVaa.sourceDeliveryProvider, "ethereum"));
  if (
    sourceDeliveryProvider !==
    ctx.deliveryProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceDeliveryProvider: deliveryVaa.sourceDeliveryProvider,
    });
    return;
  }
  processDeliveryInstruction(ctx, deliveryVaa, ctx.vaaBytes!);
}

async function processRedelivery(ctx: GRContext) {
  const redeliveryVaa = parseWormholeRelayerResend(ctx.vaa!.payload);
  const sourceDeliveryProvider = ethers.utils.getAddress(wh.tryUint8ArrayToNative(redeliveryVaa.newSourceDeliveryProvider, "ethereum"));
  if (
    sourceDeliveryProvider !==
    ctx.deliveryProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceDeliveryProvider: redeliveryVaa.newSourceDeliveryProvider,
    });
    return;
  }

  ctx.logger.info(
    `Redelivery requested for the following VAA: `,
    vaaKeyPrintable(redeliveryVaa.deliveryVaaKey)
  );

  let originalVaa = await ctx.fetchVaa(
    redeliveryVaa.deliveryVaaKey.chainId as wh.ChainId,
    Buffer.from(redeliveryVaa.deliveryVaaKey.emitterAddress!),
    redeliveryVaa.deliveryVaaKey.sequence!.toBigInt()
  );

  ctx.logger.info("Retrieved original VAA!");
  const delivery = parseWormholeRelayerSend(originalVaa.payload);
  if (!isValidRedelivery(ctx, delivery, redeliveryVaa)) {
    ctx.logger.info("Exiting redelivery process");
    return;
  } else {
    ctx.logger.info("Redelivery is valid, proceeding with redelivery");
    processDeliveryInstruction(ctx, delivery, originalVaa.bytes, {
      newReceiverValue: redeliveryVaa.newRequestedReceiverValue,
      newExecutionInfo: redeliveryVaa.newEncodedExecutionInfo,
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
  if (!redelivery.newSourceDeliveryProvider) {
  }

  const [deliveryExecutionInfo,] = parseEVMExecutionInfoV1(delivery.encodedExecutionInfo, 0);
  const [redeliveryExecutionInfo,] = parseEVMExecutionInfoV1(redelivery.newEncodedExecutionInfo, 0);

  if (deliveryExecutionInfo.targetChainRefundPerGasUnused.gt(redeliveryExecutionInfo.targetChainRefundPerGasUnused)) {
    ctx.logger.info(
      "Redelivery target chain refund per gas unused is less than original delivery target chain refund per gas unused"
    );
    ctx.logger.info(
      "Original refund: " +
        deliveryExecutionInfo.targetChainRefundPerGasUnused.toBigInt().toLocaleString() +
        " Redelivery: " +
        redeliveryExecutionInfo.targetChainRefundPerGasUnused.toBigInt().toLocaleString()
    );
    return false;
  }
  if (delivery.requestedReceiverValue.gt(redelivery.newRequestedReceiverValue)) {
    ctx.logger.info(
      "Redelivery requested receiverValue is less than original delivery requested receiverValue"
    );
    ctx.logger.info(
      "Original refund: " +
        delivery.requestedReceiverValue.toBigInt().toLocaleString(),
      +" Redelivery: " +
        redelivery.newRequestedReceiverValue.toBigInt().toLocaleString()
    );
    return false;
  }
  if (
    deliveryExecutionInfo.gasLimit >
    redeliveryExecutionInfo.gasLimit
  ) {
    ctx.logger.info(
      "Redelivery gasLimit is less than original delivery gasLimit"
    );
    ctx.logger.info(
      "Original refund: " + deliveryExecutionInfo.gasLimit,
      " Redelivery: " + redeliveryExecutionInfo.gasLimit
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
  const receiverValue = overrides?.newReceiverValue
    ? overrides.newReceiverValue
    : (delivery.requestedReceiverValue.add(delivery.extraReceiverValue));
  const getMaxRefund = (encodedDeliveryInfo: Buffer) => {
    const [deliveryInfo,] = parseEVMExecutionInfoV1(encodedDeliveryInfo, 0);
    return deliveryInfo.targetChainRefundPerGasUnused.mul(deliveryInfo.gasLimit);
  }
  const maxRefund = getMaxRefund(overrides?.newExecutionInfo
    ? overrides.newExecutionInfo
    : delivery.encodedExecutionInfo);
  const budget = receiverValue.add(maxRefund);

  await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
    const wormholeRelayer = IWormholeRelayerDelivery__factory.connect(
      ctx.wormholeRelayers[chainId],
      wallet
    );

    const encodedVMs = results.map((v) => v.bytes);
    const packedOverrides = overrides ? packOverrides(overrides) : [];
    const gasUnitsEstimate = await wormholeRelayer.estimateGas.deliver(encodedVMs, deliveryVaa, wallet.address, packedOverrides, {
      value: budget,
      gasLimit: 3000000,
    });
    const gasPrice = await wormholeRelayer.provider.getGasPrice();
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
    const receipt = await wormholeRelayer
      .deliver(encodedVMs, deliveryVaa, wallet.address, packedOverrides, { value: budget, gasLimit: 3000000 })
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
    const parsedLog = IWormholeRelayerDelivery__factory.createInterface().parseLog(
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
