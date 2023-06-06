import { ChainId } from "@certusone/wormhole-sdk";
import { BigNumber } from "ethers";

export type DeliveryRecord = {
  targetChain: ChainId;
  targetTxHash: string;
  targetTxTimestamp: number;
  txFeeTarget: BigNumber; //Total transaction fee paid, in native token
  valueTarget: BigNumber; //Total value transferred in, in native token
  relayerRefundTarget: BigNumber; //Total relayer refund, in native token
  spotQuoteTarget: number; //Price quote of native token in USD at the time of the transaction
  sourceRecord: SourceRecord; //record of the source
  netCost: BigNumber; //Total cost to the relayer, in native token
  netCostUsdSpot: number; //Total cost to the relayer, in USD
  netProfitUsdSpot: number; //Total profit to the relayer, in USD
};

export type SourceRecord = {
  sourceChain: ChainId;
  sourceTxHash?: string;
  sourceTxTimestamp?: number;
  sourceTxDidRevert: boolean;
  sourceVaaHash: string;
  sourceVaaSequence: bigint;
  isRedelivery: boolean;
  sourceCaptureNative: BigNumber; //Total amount captured, in native token
  sourceSpotQuote?: number; //Price quote of native token in USD at the time of the transaction
  sourceCaptureUsdSpot: number; //Total amount captured, in USD
};

export enum DeliveryStatus {
  WaitingForVAA = "Waiting for VAA",
  PendingDelivery = "Pending Delivery",
  DeliverySuccess = "Delivery Success",
  ReceiverFailure = "Receiver Failure",
  ForwardRequestSuccess = "Forward Request Success",
  ForwardRequestFailure = "Forward Request Failure",
  ThisShouldNeverHappen = "This should never happen. Contact Support.",
  DeliveryDidntHappenWithinRange = "Delivery didn't happen within given block range",
}

export enum RefundStatus {
  RefundSent,
  RefundFail,
  CrossChainRefundSent,
  CrossChainRefundSentMaximumBudget,
  CrossChainRefundFailProviderNotSupported,
  CrossChainRefundFailNotEnough,
}

export type DeliveryOverrideArgs = {
  gasLimit: number;
  newMaximumRefundTarget: BigNumber;
  newReceiverValueTarget: BigNumber;
  redeliveryHash: Buffer;
};

export type DeliveryTargetInfo = {
  status: DeliveryStatus | string;
  deliveryTxHash: string | null;
  vaaHash: string | null;
  sourceChain: number | null;
  sourceVaaSequence: BigNumber | null;
  gasUsed: number;
  refundStatus: RefundStatus;
  leftoverTransactionFee?: number; // Only defined if status is FORWARD_REQUEST_SUCCESS
  revertData?: string; // Only defined if status is RECEIVER_FAILURE or FORWARD_REQUEST_FAILURE
  overrides?: DeliveryOverrideArgs;
};
