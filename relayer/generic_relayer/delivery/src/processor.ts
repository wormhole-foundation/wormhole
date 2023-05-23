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
import {
  CoreRelayer__factory,
  IDelivery,
} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";
import {
  DeliveryExecutionRecord,
  addFatalError,
  deliveryExecutionRecordPrintable,
} from "./executionRecord";

export async function processGenericRelayerVaa(ctx: GRContext, next: Next) {
  const executionRecord: DeliveryExecutionRecord = {};
  executionRecord.executionStartTime = Date.now();

  try {
    ctx.logger.info(`Processing generic relayer vaa`);

    executionRecord.rawVaaHex = ctx.vaaBytes!.toString("hex");
    executionRecord.rawVaaPayloadHex = ctx.vaa!.payload.toString("hex");

    const payloadId = parseWormholeRelayerPayloadType(ctx.vaa!.payload);

    executionRecord.payloadType = RelayerPayloadId[payloadId];

    // route payload types
    if (payloadId == RelayerPayloadId.Delivery) {
      ctx.logger.info(`Detected delivery VAA, processing delivery payload...`);
      await processDelivery(ctx, executionRecord);
    } else if (payloadId == RelayerPayloadId.Redelivery) {
      ctx.logger.info(
        `Detected redelivery VAA, processing redelivery payload...`
      );
      await processRedelivery(ctx, executionRecord);
    } else {
      ctx.logger.error(`Expected GR Delivery payload type, found ${payloadId}`);
      throw new Error("Expected GR Delivery payload type");
    }

    executionRecord.didError = false;
  } catch (e: any) {
    ctx.logger.error(`Fatal error in processGenericRelayerVaa: ${e}`);
    addFatalError(executionRecord, e);
    ctx.logger.error("Dumping execution context for fatal error");
    ctx.logger.error(deliveryExecutionRecordPrintable(executionRecord));
  }

  executionRecord.executionEndTime = Date.now();
  await next();
}

async function processDelivery(
  ctx: GRContext,
  executionRecord: DeliveryExecutionRecord
) {
  const deliveryVaa = parseWormholeRelayerSend(ctx.vaa!.payload);
  executionRecord.didParse = true;
  const sourceRelayProvider = ethers.utils.getAddress(
    wh.tryUint8ArrayToNative(deliveryVaa.sourceRelayProvider, "ethereum")
  );
  if (
    sourceDeliveryProvider !==
    ctx.deliveryProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceDeliveryProvider: deliveryVaa.sourceDeliveryProvider,
    });
    executionRecord.didMatchRelayProvider = false;
    executionRecord.specifiedRelayProvider = sourceRelayProvider;
    return;
  }
  processDeliveryInstruction(ctx, deliveryVaa, ctx.vaaBytes!, executionRecord);
}

async function processRedelivery(
  ctx: GRContext,
  executionRecord: DeliveryExecutionRecord
) {
  executionRecord.redeliveryRecord = {};
  const redeliveryVaa = parseWormholeRelayerResend(ctx.vaa!.payload);
  const sourceRelayProvider = ethers.utils.getAddress(
    wh.tryUint8ArrayToNative(redeliveryVaa.sourceRelayProvider, "ethereum")
  );
  if (
    sourceDeliveryProvider !==
    ctx.deliveryProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    ctx.logger.info("Delivery vaa specifies different relay provider", {
      sourceDeliveryProvider: redeliveryVaa.newSourceDeliveryProvider,
    });
    executionRecord.didMatchRelayProvider = false;
    executionRecord.specifiedRelayProvider = sourceRelayProvider;
    return;
  }

  if (redeliveryVaa.vaaKey.payloadType != VaaKeyType.EMITTER_SEQUENCE) {
    executionRecord.redeliveryRecord.validVaaKeyFormat = false;
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }

  ctx.logger.info(
    `Redelivery requested for the following VAA: `,
    vaaKeyPrintable(redeliveryVaa.deliveryVaaKey)
  );
  executionRecord.redeliveryRecord.vaaKeyPrintable = vaaKeyPrintable(
    redeliveryVaa.vaaKey
  ).toString();

  executionRecord.redeliveryRecord.originalVaaFetchTimeStart = Date.now();

  let originalVaa: ParsedVaaWithBytes;
  try {
    originalVaa = await ctx.fetchVaa(
      redeliveryVaa.vaaKey.chainId as wh.ChainId,
      Buffer.from(redeliveryVaa.vaaKey.emitterAddress!),
      redeliveryVaa.vaaKey.sequence!.toBigInt()
    );
    executionRecord.redeliveryRecord.originalVaaDidFetch = true;
    executionRecord.redeliveryRecord.originalVaaHex =
      originalVaa.bytes.toString("hex");
  } catch (e: any) {
    //TODO this failure mode is encountered both if the VAA does not exist, I.E, the redelivery is invalid,
    // but also if there's just a network or RPC error in fetching the VAA. We should distinguish between these
    // two cases, because the first case does not need to be retried, but the second case does.
    ctx.logger.error(
      `Failed while attempting to pull original delivery VAA: ${e}`
    );
    addFatalError(executionRecord, e);
    return;
  }

  executionRecord.redeliveryRecord.originalVaaFetchTimeEnd = Date.now();

  ctx.logger.info("Retrieved original VAA!");
  const delivery = parseWormholeRelayerSend(originalVaa.payload);
  const validityCheck = isValidRedelivery(ctx, delivery, redeliveryVaa); //TODO better name?
  if (!validityCheck.isValid) {
    ctx.logger.info("Exiting redelivery process");
    executionRecord.redeliveryRecord.isValidRedelivery = false;
    executionRecord.redeliveryRecord.invalidRedeliveryReason =
      validityCheck.reason;
    return;
  } else {
    executionRecord.redeliveryRecord.isValidRedelivery = true;
    ctx.logger.info("Redelivery is valid, proceeding with redelivery");
    processDeliveryInstruction(
      ctx,
      delivery,
      originalVaa.bytes,
      executionRecord,
      {
        newReceiverValueTarget: redeliveryVaa.newReceiverValueTarget,
        newMaximumRefundTarget: redeliveryVaa.newMaximumRefundTarget,
        gasLimit: redeliveryVaa.executionParameters.gasLimit,
        redeliveryHash: ctx.vaa!.hash,
      }
    );
  }
}

function isValidRedelivery(
  ctx: GRContext,
  delivery: DeliveryInstruction,
  redelivery: RedeliveryInstruction
): { isValid: boolean; reason?: string } {
  const output: any = { isValid: true };

  //TODO check that the delivery & redelivery chains agree!
  if (delivery.targetChain != redelivery.targetChain) {
    output.isValid = false;
    output.reason =
      "Redelivery targetChain does not match original delivery targetChain, " +
      "Original targetChain: " +
      delivery.targetChain +
      " Redelivery targetChain: " +
      redelivery.targetChain;
    ctx.logger.info(output.reason);
    return output;
  }

  //TODO verify that this check is correct, and eventually move all of these style checks to be
  //uniformly done with wormhole address formats
  if (
    redelivery.sourceRelayProvider.toString("hex") !==
    ctx.relayProviders[ctx.vaa!.emitterChain as EVMChainId]
  ) {
    output.isValid = false;
    output.reason =
      "Redelivery vaa specifies different relay provider, " +
      redelivery.sourceRelayProvider.toString("hex") +
      " vs " +
      ctx.relayProviders[ctx.vaa!.emitterChain as EVMChainId];
    ctx.logger.info(output.reason);
    return output;
  }

  if (delivery.maximumRefundTarget.gt(redelivery.newMaximumRefundTarget)) {
    output.isValid = false;
    output.reason =
      "Redelivery maximumRefundTarget is less than original delivery maximumRefundTarget, " +
      "Original refund: " +
      delivery.maximumRefundTarget.toBigInt().toLocaleString() +
      " Redelivery: " +
      redelivery.newMaximumRefundTarget.toBigInt().toLocaleString();
    ctx.logger.info(output.reason);
    return output;
  }
  if (delivery.receiverValueTarget.gt(redelivery.newReceiverValueTarget)) {
    output.isValid = false;
    (output.reason =
      "Redelivery receiverValueTarget is less than original delivery receiverValueTarget, " +
      "Original receiverValue: " +
      delivery.receiverValueTarget.toBigInt().toLocaleString()),
      +" Redelivery: " +
        redelivery.newReceiverValueTarget.toBigInt().toLocaleString();
    ctx.logger.info(output.reason);
    return output;
  }

  if (
    deliveryExecutionInfo.gasLimit >
    redeliveryExecutionInfo.gasLimit
  ) {
    output.isValid = false;
    (output.reason =
      "Redelivery gasLimit is less than original delivery gasLimit, " +
      "Original gasLimit: " +
      delivery.executionParameters.gasLimit),
      " Redelivery: " + redelivery.executionParameters.gasLimit;
    ctx.logger.info(output.reason);
    return output;
  }

  return output;
}

async function processDeliveryInstruction(
  ctx: GRContext,
  delivery: DeliveryInstruction,
  deliveryVaa: Buffer | Uint8Array,
  executionRecord: DeliveryExecutionRecord,
  overrides?: DeliveryOverrideArgs
) {
  executionRecord.deliveryRecord = {};
  executionRecord.deliveryRecord.deliveryInstructionsPrintable =
    deliveryInstructionsPrintable(delivery).toString();
  executionRecord.deliveryRecord.hasAdditionalVaas =
    delivery.vaaKeys.length > 0;

  //TODO this check is not quite correct
  if (
    delivery.vaaKeys.findIndex(
      (m) => m.infoType !== VaaKeyType.EMITTER_SEQUENCE
    ) != -1
  ) {
    executionRecord.deliveryRecord.additionalVaaKeysFormatValid = false;
    throw new Error(`Only supports EmitterSequence VaaKeyType`);
  }
  const vaaKeysString = delivery.vaaKeys.map((m) => vaaKeyPrintable(m));
  ctx.logger.info(`Fetching vaas from parsed delivery vaa manifest...`, {
    vaaKeys: vaaKeysString,
  });
  executionRecord.deliveryRecord.additionalVaaKeysPrintable =
    vaaKeysString.toString();

  const vaaIds = delivery.vaaKeys.map((m) => ({
    emitterAddress: m.emitterAddress!,
    emitterChain: m.chainId! as wh.ChainId,
    sequence: m.sequence!.toBigInt(),
  }));

  let results: ParsedVaaWithBytes[];

  executionRecord.deliveryRecord.fetchAdditionalVaasTimeStart = Date.now();
  try {
    results = await ctx.fetchVaas({
      ids: vaaIds,
      // txHash: ctx.sourceTxHash,
    });
    executionRecord.deliveryRecord.additionalVaasDidFetch = true;
  } catch (e: any) {
    ctx.logger.error(`Failed while attempting to pull additional VAAs: ${e}`);
    executionRecord.deliveryRecord.additionalVaasDidFetch = false;
    addFatalError(executionRecord, e);
    return;
  }
  executionRecord.deliveryRecord.fetchAdditionalVaasTimeEnd = Date.now();
  executionRecord.deliveryRecord.additionalVaasHex = results.map((v) =>
    v.bytes.toString("hex")
  );

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

  executionRecord.deliveryRecord.chainId = chainId;
  executionRecord.deliveryRecord.receiverValue = receiverValue.toString();
  executionRecord.deliveryRecord.maxRefund = maxRefund.toString();
  executionRecord.deliveryRecord.budget = budget.toString();

  executionRecord.deliveryRecord.walletAcquisitionStartTime = Date.now();

  const closedFunction = async (executionRecord: DeliveryExecutionRecord) => {
    await ctx.wallets.onEVM(chainId, async ({ wallet }) => {
      executionRecord.deliveryRecord!.walletAcquisitionEndTime = Date.now();
      executionRecord.deliveryRecord!.walletAcquisitionDidSucceed = true;
      executionRecord.deliveryRecord!.walletAddress = wallet.address;
      executionRecord.deliveryRecord!.walletBalance = (
        await wallet.getBalance()
      ).toString();
      executionRecord.deliveryRecord!.walletNonce =
        await wallet.getTransactionCount();

      const coreRelayer = CoreRelayer__factory.connect(
        ctx.wormholeRelayers[chainId],
        wallet
      );

      const input: IDelivery.TargetDeliveryParametersStruct = {
        encodedVMs: results.map((v) => v.bytes),
        encodedDeliveryVAA: deliveryVaa,
        relayerRefundAddress: wallet.address,
        overrides: overrides ? packOverrides(overrides) : [],
      };

      const gasUnitsEstimate = await coreRelayer.estimateGas.deliver(input, {
        value: budget,
        gasLimit: 3000000,
      });
      const gasPrice = await coreRelayer.provider.getGasPrice();
      const estimatedTransactionFee = gasPrice.mul(gasUnitsEstimate);
      const estimatedTransactionFeeEther = ethers.utils.formatEther(
        estimatedTransactionFee
      );

      executionRecord.deliveryRecord!.gasUnitsEstimate =
        gasUnitsEstimate.toNumber();
      executionRecord.deliveryRecord!.gasPriceEstimate = gasPrice.toString();
      executionRecord.deliveryRecord!.estimatedTransactionFee =
        estimatedTransactionFee.toString();
      executionRecord.deliveryRecord!.estimatedTransactionFeeEther =
        estimatedTransactionFeeEther;

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

      executionRecord.deliveryRecord!.transactionSubmitTimeStart = Date.now();
      const receipt = await coreRelayer
        .deliver(input, { value: budget, gasLimit: 3000000 }) //TODO more intelligent gas limit
        .then((x: any) => x.wait());
      executionRecord.deliveryRecord!.transactionSubmitTimeEnd = Date.now();
      executionRecord.deliveryRecord!.transactionDidSubmit = true;
      executionRecord.deliveryRecord!.transactionHashes = [
        receipt.transactionHash,
      ];

      logResults(ctx, receipt, chainId, executionRecord);
    });
  };

  await closedFunction(executionRecord);
}

function logResults(
  ctx: GRContext,
  receipt: ethers.ContractReceipt,
  chainId: EVMChainId,
  executionRecord: DeliveryExecutionRecord
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
    executionRecord.deliveryRecord!.resultLogDidParse = true;
    switch (logArgs.status) {
      case 0:
        ctx.logger.info("Delivery Success");
        executionRecord.deliveryRecord!.resultLog = "Delivery Success";
        break;
      case 1:
        ctx.logger.info("Receiver Failure");
        executionRecord.deliveryRecord!.resultLog = "Receiver Failure";
        break;
      case 2:
        ctx.logger.info("Forwarding Failure");
        executionRecord.deliveryRecord!.resultLog = "Forwarding Failure";
        break;
    }
  }
  ctx.logger.info(
    `Relayed instruction to chain ${chainId}, tx hash: ${receipt.transactionHash}`
  );
}
