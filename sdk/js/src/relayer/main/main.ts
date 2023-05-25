import {
  ChainId,
  CHAIN_ID_TO_NAME,
  ChainName,
  isChain,
  CONTRACTS,
  CHAINS,
  tryNativeToHexString,
  tryHexToNativeString,
  Network,
  ethers_contracts,
  getSignedVAAWithRetry,
  parseVaa,
} from "../../";
import { BigNumber, ethers } from "ethers";
import { getWormholeRelayer, getWormholeRelayerAddress } from "../consts";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  VaaKeyType,
  DeliveryStatus,
  VaaKey,
  parseWormholeRelayerSend,
  RefundStatus,
  parseEVMExecutionInfoV1
} from "../structs";
import {
  getDefaultProvider,
  printChain,
  getWormholeRelayerLog,
  parseWormholeLog,
  getBlockRange,
  getWormholeRelayerInfoBySourceSequence,
  vaaKeyToVaaKeyStruct,
  getRelayProvider,
  DeliveryTargetInfo
} from "./helpers";
import { VaaKeyStruct } from "../../ethers-contracts/IWormholeRelayer.sol/IWormholeRelayer";

export type InfoRequestParams = {
  environment?: Network;
  sourceChainProvider?: ethers.providers.Provider;
  targetChainProviders?: Map<ChainName, ethers.providers.Provider>;
  targetChainBlockRanges?: Map<
    ChainName,
    [ethers.providers.BlockTag, ethers.providers.BlockTag]
  >;
  coreRelayerWhMessageIndex?: number;
  coreRelayerAddresses?: Map<ChainName, string>
};

export type DeliveryInfo = {
  type: RelayerPayloadId.Delivery;
  sourceChain: ChainName;
  sourceTransactionHash: string;
  sourceDeliverySequenceNumber: number;
  deliveryInstruction: DeliveryInstruction;
  targetChainStatus: {
    chain: ChainName;
    events: DeliveryTargetInfo[];
  };
};

export function printWormholeRelayerInfo(info: DeliveryInfo) {
  console.log(stringifyWormholeRelayerInfo(info));
}

export function stringifyWormholeRelayerInfo(info: DeliveryInfo): string {
  let stringifiedInfo = "";
  if (info.type == RelayerPayloadId.Delivery && info.deliveryInstruction.targetAddress.toString("hex") !== "0000000000000000000000000000000000000000000000000000000000000000") {
    stringifiedInfo += `Found delivery request in transaction ${
      info.sourceTransactionHash
    } on ${info.sourceChain}\n`;
    const numMsgs = info.deliveryInstruction.vaaKeys.length;

    const payload = info.deliveryInstruction.payload.toString("hex");
    if(payload.length > 0) {
      stringifiedInfo += `\nPayload to be relayed (as hex string): 0x${payload}`
    }
    if(numMsgs > 0) {
      stringifiedInfo += `\nThe following ${numMsgs} wormhole messages (VAAs) were ${payload.length > 0 ? 'also ' : ''}requested to be relayed:\n`;
      stringifiedInfo += info.deliveryInstruction.vaaKeys.map((msgInfo, i) => {
        let result = "";
        result += `(VAA ${i}): `;
          result += `Message from ${
            msgInfo.chainId ? printChain(msgInfo.chainId) : ""
          }, with emitter address ${msgInfo.emitterAddress?.toString(
            "hex"
          )} and sequence number ${msgInfo.sequence}`;
        
        return result;
      }).join(",\n");
    }
    if(payload.length == 0 && numMsgs == 0) {
      stringifiedInfo += `\nAn empty payload was requested to be sent`
    }

    const instruction = info.deliveryInstruction;
    const targetChainName = CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];
    stringifiedInfo += `${numMsgs == 0 ? (payload.length == 0 ? '' : '\n\nPayload was requested to be relayed') : '\n\nThese were requested to be sent'} to 0x${instruction.targetAddress.toString(

      "hex"
    )} on ${printChain(instruction.targetChainId)}\n`;
    const totalReceiverValue = (instruction.requestedReceiverValue.add(instruction.extraReceiverValue));
    stringifiedInfo += totalReceiverValue.gt(0)
      ? `Amount to pass into target address: ${totalReceiverValue} wei of ${targetChainName} currency ${instruction.extraReceiverValue.gt(0) ? `${instruction.requestedReceiverValue} requested, ${instruction.extraReceiverValue} additionally paid for` : ""}\n`
      : ``;
    const [executionInfo,] = parseEVMExecutionInfoV1(instruction.encodedExecutionInfo, 0);
    stringifiedInfo += `Gas limit: ${executionInfo.gasLimit} ${targetChainName} gas\n\n`;
    stringifiedInfo += `Refund rate: ${executionInfo.targetChainRefundPerGasUnused} of ${targetChainName} wei per unit of gas unused\n\n`;
    stringifiedInfo += info.targetChainStatus.events

            .map(
              (e, i) =>
                `Delivery attempt ${i + 1}: ${
                  e.transactionHash
                    ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
                    : ""
                }\nStatus: ${e.status}\n${e.revertString ? `Failure reason: ${e.gasUsed == executionInfo.gasLimit ? "Gas limit hit" : e.revertString}\n`: ""}Gas used: ${e.gasUsed}\nTransaction fee used: ${executionInfo.targetChainRefundPerGasUnused.mul(e.gasUsed).toString()} wei of ${targetChainName} currency\n}`
            )
            .join("\n");
   } else if (info.type == RelayerPayloadId.Delivery && info.deliveryInstruction.targetAddress.toString("hex") === "0000000000000000000000000000000000000000000000000000000000000000") {
    stringifiedInfo += `Found delivery request in transaction ${
      info.sourceTransactionHash
    } on ${info.sourceChain}\n`;

    const instruction = info.deliveryInstruction;
    const targetChainName = CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];
    
    stringifiedInfo += `\nA refund of ${instruction.extraReceiverValue} ${targetChainName} wei was requested to be sent to ${targetChainName}, address 0x${info.deliveryInstruction.refundAddress.toString("hex")}`
    
    stringifiedInfo += info.targetChainStatus.events

            .map(
              (e, i) =>
                `Delivery attempt ${i + 1}: ${
                  e.transactionHash
                    ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
                    : ""
                }\nStatus: ${e.refundStatus == RefundStatus.RefundSent ? "Refund Successful" : "Refund Failed"}`
            )
            .join("\n");
   } 
   
  return stringifiedInfo;
}

export type SendOptionalParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  paymentForExtraReceiverValue?: ethers.BigNumberish;
  additionalVaas?: [
    {
      chainId?: ChainId;
      emitterAddress: string;
      sequenceNumber: ethers.BigNumberish;
    }
  ];
  relayProviderAddress?: string;
  consistencyLevel?: ethers.BigNumberish;
  refundChainId?: ChainId;
  refundAddress?: string;
  relayParameters?: ethers.BytesLike;
};

export async function sendToEvm(
  signer: ethers.Signer,
  sourceChain: ChainName,
  targetChain: ChainName,
  targetAddress: string,
  payload: ethers.BytesLike,
  gasLimit: number,
  overrides?: ethers.PayableOverrides,
  sendOptionalParams?: SendOptionalParams,
): Promise<ethers.providers.TransactionResponse> {
  const sourceChainId = CHAINS[sourceChain];
  const targetChainId = CHAINS[targetChain];

  const environment = sendOptionalParams?.environment || "MAINNET";
  const coreRelayerAddress = getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  const sourceCoreRelayer = ethers_contracts.IWormholeRelayer__factory.connect(
    coreRelayerAddress,
    signer
  );

  const refundLocationExists =
    sendOptionalParams?.refundChainId!== undefined &&
    sendOptionalParams?.refundAddress !== undefined;
  const defaultRelayProviderAddress =
    await sourceCoreRelayer.getDefaultRelayProvider();

  // Using the most general 'send' function in IWormholeRelayer
  // Inputs:
  // targetChainId, targetAddress, refundChainId, refundAddress, maxTransactionFee, receiverValue, payload, vaaKeys, 
  // consistencyLevel, relayProviderAddress, relayParameters 
  const [deliveryPrice,]: [BigNumber, BigNumber] = await sourceCoreRelayer["quoteEVMDeliveryPrice(uint16,uint128,uint32,address)"](targetChainId, sendOptionalParams?.receiverValue || 0, gasLimit, sendOptionalParams?.relayProviderAddress || defaultRelayProviderAddress);
  const value = await (overrides?.value || 0);
  const totalPrice = deliveryPrice.add(sendOptionalParams?.paymentForExtraReceiverValue || 0);
  if(!totalPrice.eq(value)) {
    throw new Error(`Expected a payment of ${totalPrice.toString()} wei; received ${value.toString()} wei`);
  }
  const tx = sourceCoreRelayer["sendToEvm(uint16,address,bytes,uint256,uint256,uint256,uint16,address,address,(uint16,bytes32,uint64)[],uint8)"](
    targetChainId, // targetChainId
    targetAddress, // targetAddress
    payload,
    sendOptionalParams?.receiverValue || 0, // receiverValue 
    sendOptionalParams?.paymentForExtraReceiverValue || 0, // payment for extra receiverValue 
    gasLimit,
    (refundLocationExists && sendOptionalParams?.refundChainId) || sourceChainId, // refundChainId
    refundLocationExists &&
          sendOptionalParams?.refundAddress &&
          sendOptionalParams?.refundAddress ||
          signer.getAddress(), // refundAddress
    sendOptionalParams?.relayProviderAddress || defaultRelayProviderAddress, // relayProviderAddress
    sendOptionalParams?.additionalVaas
      ? sendOptionalParams.additionalVaas.map(
          (additionalVaa): VaaKeyStruct => ({
            chainId: additionalVaa.chainId || sourceChainId,
            emitterAddress: Buffer.from(tryNativeToHexString(
              additionalVaa.emitterAddress,
              "ethereum"
            ), "hex"),
            sequence: BigNumber.from(additionalVaa.sequenceNumber || 0)
          })
        )
      : [], // vaaKeys
    sendOptionalParams?.consistencyLevel || 15, // consistencyLevel
  overrides);
  return tx;
}

export type GetPriceOptParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  relayProviderAddress?: string;
  sourceChainProvider?: ethers.providers.Provider;
};

export async function getPriceAndRefundInfo(
  sourceChain: ChainName,
  targetChain: ChainName,
  gasAmount: ethers.BigNumberish,
  optionalParams?: GetPriceOptParams
): Promise<[ethers.BigNumber, ethers.BigNumber]> {
  const environment = optionalParams?.environment || "MAINNET";
  const sourceChainProvider =
    optionalParams?.sourceChainProvider ||
    getDefaultProvider(environment, sourceChain);
  if (!sourceChainProvider)
    throw Error(
      "No default RPC for this chain; pass in your own provider (as sourceChainProvider)"
    );
  const coreRelayerAddress = getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  const sourceCoreRelayer = ethers_contracts.IWormholeRelayer__factory.connect(
    coreRelayerAddress,
    sourceChainProvider
  );
  const relayProviderAddress =
    optionalParams?.relayProviderAddress ||
    (await sourceCoreRelayer.getDefaultRelayProvider());
  const targetChainId = CHAINS[targetChain];
  const priceAndRefundInfo = (
    await sourceCoreRelayer["quoteEVMDeliveryPrice(uint16,uint128,uint32,address)"](
      targetChainId,
      optionalParams?.receiverValue || 0,
      gasAmount,
      relayProviderAddress
    )
  )
  return priceAndRefundInfo;
}

export async function getPrice(
  sourceChain: ChainName,
  targetChain: ChainName,
  gasAmount: ethers.BigNumberish,
  optionalParams?: GetPriceOptParams
): Promise<ethers.BigNumber> {
  const priceAndRefundInfo = await getPriceAndRefundInfo(sourceChain, targetChain, gasAmount, optionalParams);
  return priceAndRefundInfo[0];
}


export async function getWormholeRelayerInfo(
  sourceChain: ChainName,
  sourceTransaction: string,
  infoRequest?: InfoRequestParams
): Promise<DeliveryInfo> {
  const environment = infoRequest?.environment || "MAINNET";
  const sourceChainProvider =
    infoRequest?.sourceChainProvider ||
    getDefaultProvider(environment, sourceChain);
  if (!sourceChainProvider)
    throw Error(
      "No default RPC for this chain; pass in your own provider (as sourceChainProvider)"
    );
  const receipt = await sourceChainProvider.getTransactionReceipt(
    sourceTransaction
  );
  if (!receipt) throw Error("Transaction has not been mined");
  const bridgeAddress =
    CONTRACTS[environment][sourceChain].core;
  const coreRelayerAddress = infoRequest?.coreRelayerAddresses?.get(sourceChain) || getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  if (!bridgeAddress || !coreRelayerAddress) {
    throw Error(
      `Invalid chain ID or network: Chain ${sourceChain}, ${environment}`
    );
  }
  const deliveryLog = getWormholeRelayerLog(
    receipt,
    bridgeAddress,
    tryNativeToHexString(coreRelayerAddress, "ethereum"),
    infoRequest?.coreRelayerWhMessageIndex
      ? infoRequest.coreRelayerWhMessageIndex
      : 0
  );

  const { type, parsed } = parseWormholeLog(deliveryLog.log);

  const instruction = parsed as DeliveryInstruction;

  const targetChainId = instruction.targetChainId as ChainId;
  if (!isChain(targetChainId)) throw Error(`Invalid Chain: ${targetChainId}`);
  const targetChain = CHAIN_ID_TO_NAME[targetChainId];
  const targetChainProvider =
    infoRequest?.targetChainProviders?.get(targetChain) ||
    getDefaultProvider(environment, targetChain);

  if (!targetChainProvider) {
    throw Error(
      "No default RPC for this chain; pass in your own provider (as targetChainProvider)"
    );
  }
  const [blockStartNumber, blockEndNumber] =
    infoRequest?.targetChainBlockRanges?.get(targetChain) ||
    getBlockRange(targetChainProvider);

    const targetChainStatus = await getWormholeRelayerInfoBySourceSequence(
      environment,
      targetChain,
      targetChainProvider,
      sourceChain,
      BigNumber.from(deliveryLog.sequence),
      blockStartNumber,
      blockEndNumber,
      infoRequest?.coreRelayerAddresses?.get(targetChain) || getWormholeRelayerAddress(
        targetChain,
        environment
      )
    );

    return {
      type: RelayerPayloadId.Delivery,
      sourceChain: sourceChain,
      sourceTransactionHash: sourceTransaction,
      sourceDeliverySequenceNumber: BigNumber.from(
        deliveryLog.sequence
      ).toNumber(),
      deliveryInstruction: instruction,
      targetChainStatus,
    };

  
}

export async function resendRaw(
  signer: ethers.Signer,
  sourceChain: ChainName,
  targetChain: ChainName,
  environment: Network,
  vaaKey: VaaKey,
  newGasLimit: number,
  newReceiverValue: BigNumber | number,
  relayProviderAddress: string,
  overrides?: ethers.PayableOverrides
) {
  const provider = signer.provider;

  if (!provider) throw Error("No provider on signer");

  const coreRelayer = getWormholeRelayer(sourceChain, environment, signer);

  return coreRelayer.resendToEvm(
    vaaKeyToVaaKeyStruct(vaaKey),
    CHAINS[targetChain],
    newReceiverValue,
    newGasLimit,
    relayProviderAddress,
    overrides
  );
}

export async function resend(
  signer: ethers.Signer,
  sourceChain: ChainName,
  targetChain: ChainName,
  environment: Network,
  vaaKey: VaaKey,
  newGasLimit: number,
  newReceiverValue: BigNumber | number,
  relayProviderAddress: string,
  wormholeRPCs: string[],
  overrides: ethers.PayableOverrides,
  isNode?: boolean,
) {
  const sourceChainId = CHAINS[sourceChain];
  const targetChainId = CHAINS[targetChain];
  const originalVAA = await getVAA(wormholeRPCs, vaaKey, isNode);

  if (!originalVAA) throw Error("orignal VAA not found");

  const originalVAAparsed = parseWormholeRelayerSend(
    parseVaa(Buffer.from(originalVAA)).payload
  );
  if (!originalVAAparsed) throw Error("orignal VAA not a valid delivery VAA.");

  const [originalExecutionInfo,] = parseEVMExecutionInfoV1(originalVAAparsed.encodedExecutionInfo, 0);
  const originalGasLimit = originalExecutionInfo.gasLimit;
  const originalRefund = originalExecutionInfo.targetChainRefundPerGasUnused;
  const originalReceiverValue = originalVAAparsed.requestedReceiverValue;
  const originalTargetChain = originalVAAparsed.targetChainId;

  

  if (originalTargetChain != targetChainId) {
    throw Error(
      `Target chain of original VAA (${originalTargetChain}) does not match target chain of resend (${targetChainId})`
    );
  }

  if (newReceiverValue < originalReceiverValue) {
    throw Error(
      `New receiver value too low. Minimum is ${originalReceiverValue.toString()}`
    );
  }

  if (newGasLimit < originalGasLimit) {
    throw Error(
      `New gas limit too low. Minimum is ${originalReceiverValue.toString()}`
    );
  }

  

  const coreRelayer = getWormholeRelayer(sourceChain, environment, signer);
  const relayProvider = getRelayProvider(
    relayProviderAddress,
    signer.provider!
  );

  const [deliveryPrice, refundPerUnitGas]: [BigNumber, BigNumber] = await coreRelayer["quoteEVMDeliveryPrice(uint16,uint128,uint32,address)"](targetChainId, newReceiverValue || 0, newGasLimit, relayProviderAddress);
  const value = await (overrides?.value || 0);
  if(!deliveryPrice.eq(value)) {
    throw new Error(`Expected a payment of ${deliveryPrice.toString()} wei; received ${value.toString()} wei`);
  }


  if (refundPerUnitGas < originalRefund) {
    throw Error(
      `New refund per unit gas too low. Minimum is ${originalRefund.toString()}.`
    );
  }

  return resendRaw(
    signer,
    sourceChain,
    targetChain,
    environment,
    vaaKey,
    newGasLimit,
    newReceiverValue,
    relayProviderAddress,
    overrides
  );
}

export async function getVAA(
  wormholeRPCs: string[],
  vaaKey: VaaKey,
  isNode?: boolean
): Promise<Uint8Array> {

  const vaa = await getSignedVAAWithRetry(
    wormholeRPCs,
    vaaKey.chainId! as ChainId,
    vaaKey.emitterAddress!.toString("hex"),
    vaaKey.sequence!.toBigInt().toString(),
    isNode
      ? {
          transport: NodeHttpTransport(),
        }
      : {},
    2000,
    4
  );

  return vaa.vaaBytes;
}
