export type DeliveryExecutionRecord = {
  didError?: boolean; // if true, the error will be logged in fatalStackTrace
  errorName?: string; // If a detectable error occurred, this is the name of that failure.
  didSubmitTransaction?: boolean; // if true, the process submitted at least one transaction, which will be logged in transactionHashes

  executionStartTime?: number; // unix timestamp in milliseconds
  executionEndTime?: number; // unix timestamp in milliseconds

  rawVaaHex?: string; // hex string of the raw VAA
  rawVaaPayloadHex?: string; // hex string of the raw VAA payload
  payloadType?: string; // the payload type of the VAA
  didParse?: boolean; // if true, the VAA was successfully parsed
  specifiedDeliveryProvider?: string; // the relay provider specified in the VAA
  didMatchDeliveryProvider?: boolean; // if true, the relay provider specified in the VAA matched the relay provider for the chain

  redeliveryRecord?: RedeliveryRecord; // if the VAA is a redelivery, the redeliveryRecord
  deliveryRecord?: DeliveryRecord; // information about the delivery process of the VAA

  fatalStackTrace?: string; // if the top level unexpected exception try-catch caught, this was the stack trace
};

export type RedeliveryRecord = {
  validVaaKeyFormat?: boolean; // if true, the VAA key format interpretable
  vaaKeyPrintable?: string; // the VAA key in printable format of the original VAA
  originalVaaFetchTimeStart?: number; // unix timestamp in milliseconds
  originalVaaFetchTimeEnd?: number; // unix timestamp in milliseconds
  originalVaaDidFetch?: boolean; // if true, the original VAA was successfully fetched
  originalVaaHex?: string; // hex string of the original VAA
  originalVaaDidParse?: boolean; // if true, the original VAA was successfully parsed
  isValidRedelivery?: boolean; // if true, the redelivery VAA is valid
  invalidRedeliveryReason?: string; // if the redelivery VAA is invalid, the reason why
};

export type DeliveryRecord = {
  deliveryInstructionsPrintable?: string; // the delivery instructions in printable format
  hasAdditionalVaas?: boolean; // if true, the delivery instructions contain additional VAAs
  additionalVaaKeysFormatValid?: boolean; // if true, the additional VAA key format interpretable
  additionalVaaKeysPrintable?: string; // the additional VAA key in printable format
  fetchAdditionalVaasTimeStart?: number; // unix timestamp in milliseconds
  fetchAdditionalVaasTimeEnd?: number; // unix timestamp in milliseconds
  additionalVaasDidFetch?: boolean; // if true, the additional VAAs were successfully fetched
  additionalVaasHex?: string[]; // hex string of the additional VAAs
  chainId?: number; // the chain ID of the chain the VAA is being sent to
  receiverValue?: string; // the receiver value of the VAA;
  maxRefund?: string; // the max refund of the VAA;
  budget?: string; // the budget of the VAA;
  walletAcquisitionStartTime?: number; // unix timestamp in milliseconds
  walletAcquisitionEndTime?: number; // unix timestamp in milliseconds
  walletAcquisitionDidSucceed?: boolean; // if true, the wallet acquisition was successful
  walletAddress?: string; // the wallet address of the wallet used to send the VAA
  walletBalance?: string; // the balance of the wallet used to send the VAA
  walletNonce?: number; // the nonce of the wallet used to send the VAA
  gasUnitsEstimate?: number; // the gas units estimate for the transaction being submitted
  gasPriceEstimate?: string; // the gas price estimate for the transaction being submitted
  estimatedTransactionFee?: string; // the estimated transaction fee for the transaction being submitted
  estimatedTransactionFeeEther?: string; // the estimated transaction fee for the transaction being submitted in the base units of the chain
  transactionSubmitTimeStart?: number; // unix timestamp in milliseconds
  transactionSubmitTimeEnd?: number; // unix timestamp in milliseconds
  transactionDidSubmit?: boolean; // if true, the transaction was successfully submitted
  transactionHashes?: string[]; // the transaction hashes of the transactions submitted
  resultLogDidParse?: boolean; // if true, the result log was successfully parsed
  resultLog?: string; // the result log of the transaction
};

export function deliveryExecutionRecordPrintable(
  executionRecord: DeliveryExecutionRecord
): string {
  return JSON.stringify(executionRecord, null, 2); //TODO deal with line breaks and such better
}

export function addFatalError(
  executionRecord: DeliveryExecutionRecord,
  e: any
) {
  executionRecord.didError = true;
  executionRecord.errorName = e.name;
  executionRecord.fatalStackTrace = e.stack
    ? e.stack.replace(/\n/g, "\\n")
    : "";
}
