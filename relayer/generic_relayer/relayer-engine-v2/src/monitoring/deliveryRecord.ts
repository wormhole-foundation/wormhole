import { ChainId } from "@certusone/wormhole-sdk";
import { BigNumber, ethers } from "ethers";
import { ParsedVaaWithBytes } from "relayer-engine";
import {
  DeliveryInstruction,
  RedeliveryInstruction,
  RelayerPayloadId,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerResend,
  parseWormholeRelayerSend,
} from "../../pkgs/sdk/src";
import {
  SourceRecord,
  DeliveryRecord,
  DeliveryTargetInfo,
  RefundStatus,
  DeliveryStatus,
} from "./consts";

async function getOriginatingTransaction(
  vaa: ParsedVaaWithBytes
): Promise<ethers.providers.TransactionReceipt> {
  //TODO this:
  //either by combing the blockchain history or using a third party indexed API
  throw new Error("Not implemented");
}

function getCaptureAmount(
  vaa: ParsedVaaWithBytes,
  tx: ethers.providers.TransactionReceipt
): BigNumber {
  //TODO this:
  //Parse events off the transaction,
  //Find corresponding payment event for this specific VAA,
  //return delivery amount
  throw new Error("Not implemented");
}

function getProvider(chain: ChainId): ethers.providers.Provider {
  //TODO this:
  //return a provider for the specified chain
  throw new Error("Not implemented");
}

async function quoteNativeToken(
  chain: ChainId,
  timestamp: number
): Promise<number> {
  //TODO this:
  //return the price of the native token on the specified chain at the specified time
  throw new Error("Not implemented");
}

async function getRelayerRefundAmount(
  vaa: ParsedVaaWithBytes,
  deliveryTx: ethers.providers.TransactionReceipt
): Promise<BigNumber> {
  //TODO: Find corresponding delivery event
  let deliveryEvent: DeliveryTargetInfo = {} as any;
  const receiverValueWasPaid =
    deliveryEvent.status != DeliveryStatus.ReceiverFailure &&
    deliveryEvent.status != DeliveryStatus.ForwardRequestFailure;

  //parse the VAA for the maximum refund amount
  const deliveryFields = await parseDeliveryFields(vaa);

  const gasUtilization = BigNumber.from(
    BigInt(deliveryEvent.gasUsed) / BigInt(deliveryFields.gasLimit)
  );
  const maxFeeUtilization: BigNumber = gasUtilization.mul(
    deliveryFields.maximumRefund
  );

  const receiverValueRefundAmount = receiverValueWasPaid
    ? deliveryFields.receiverValue
    : BigNumber.from(0);

  const refundToRefundAddress = receiverValueRefundAmount.add(
    deliveryFields.maximumRefund.sub(maxFeeUtilization)
  );

  const relayerRefundAmount = BigNumber.from(0) //assume the relayer did not put in additional funds
    .add(maxFeeUtilization) //Utilized fees are returned
    .add(
      deliveryEvent.refundStatus == RefundStatus.RefundFail
        ? refundToRefundAddress
        : BigNumber.from(0)
    );

  throw new Error("Not implemented");
  //return relayerRefundAmount;
}

async function getTime(
  chain: ChainId,
  tx: ethers.providers.TransactionReceipt
): Promise<number> {
  const provider = getProvider(chain);
  return (await provider.getBlock(tx.blockNumber)).timestamp;
}

async function createSourceRecord(
  vaa: ParsedVaaWithBytes
): Promise<SourceRecord> {
  const tx = await getOriginatingTransaction(vaa);
  const time = !!tx && (await getTime(vaa.emitterChain as ChainId, tx));

  const sourceCaptureNative = !!tx
    ? getCaptureAmount(vaa, tx)
    : BigNumber.from(0);
  const sourceSpotQuote = !!tx
    ? await quoteNativeToken(vaa.emitterChain as ChainId, time)
    : undefined;
  const sourceCaptureUsdSpot = !!tx
    ? sourceCaptureNative.mul(sourceSpotQuote!).toNumber()
    : 0;

  return {
    sourceChain: vaa.emitterChain as ChainId,
    sourceTxDidRevert: !!tx,
    sourceTxHash: !!tx && tx.transactionHash,
    sourceTxTimestamp: time,
    sourceVaaHash: vaa.hash.toString("hex"),
    sourceVaaSequence: vaa.sequence,
    isRedelivery:
      parseWormholeRelayerPayloadType(vaa.payload) ==
      RelayerPayloadId.Redelivery,
    sourceCaptureNative,
    sourceSpotQuote,
    sourceCaptureUsdSpot,
  };
}

async function createDeliveryRecord(
  vaa: ParsedVaaWithBytes,
  deliveryTx: ethers.providers.TransactionReceipt
): Promise<DeliveryRecord> {
  const sourceRecord = await createSourceRecord(vaa);
  const deliveryFields = await parseDeliveryFields(vaa);
  const time = await getTime(deliveryFields.targetChain, deliveryTx);
  const valueTarget = deliveryFields.receiverValue.add(
    deliveryFields.maximumRefund
  );
  const relayerRefundTarget = await getRelayerRefundAmount(vaa, deliveryTx);

  const txFeeTarget = deliveryTx.effectiveGasPrice.mul(deliveryTx.gasUsed);
  const netCost = txFeeTarget.add(valueTarget).sub(relayerRefundTarget);
  const spotQuoteTarget = await quoteNativeToken(
    deliveryFields.targetChain,
    time
  );
  const netCostUsdSpot = netCost.mul(spotQuoteTarget).toNumber();

  return {
    targetChain: deliveryFields.targetChain,
    targetTxHash: deliveryTx.transactionHash,
    targetTxTimestamp: time,
    txFeeTarget,
    valueTarget, //Total value transferred in, in native token
    relayerRefundTarget, //Total relayer refund, in native token
    spotQuoteTarget, //Price quote of native token in USD at the time of the transaction
    sourceRecord, //record of the source,
    netCost,
    netCostUsdSpot,
    netProfitUsdSpot: sourceRecord.sourceCaptureUsdSpot - netCostUsdSpot,
  };
}

export type DeliveryFields = {
  targetChain: ChainId;
  maximumRefund: BigNumber;
  receiverValue: BigNumber;
  gasLimit: number;
};

function parseDeliveryFields(vaa: ParsedVaaWithBytes): DeliveryFields {
  const payloadId = parseWormholeRelayerPayloadType(vaa.payload);
  let redelivery: RedeliveryInstruction | undefined;
  let delivery: DeliveryInstruction | undefined;
  if (payloadId == RelayerPayloadId.Delivery) {
    delivery = parseWormholeRelayerSend(vaa.payload);
  } else if (payloadId == RelayerPayloadId.Redelivery) {
    redelivery = parseWormholeRelayerResend(vaa.payload);
  } else {
    throw new Error("Specified VAA is not a delivery or redelivery VAA");
  }
  return {
    targetChain: (delivery
      ? delivery.targetChain
      : redelivery!.targetChain) as ChainId,
    maximumRefund: delivery
      ? delivery.maximumRefundTarget
      : redelivery!.newMaximumRefundTarget,
    receiverValue: delivery
      ? delivery.receiverValueTarget
      : redelivery!.newReceiverValueTarget,
    gasLimit: delivery
      ? delivery.executionParameters.gasLimit
      : redelivery!.executionParameters.gasLimit,
  };
}
