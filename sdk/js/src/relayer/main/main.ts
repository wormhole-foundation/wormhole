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
        if (msgInfo.infoType == VaaKeyType.EMITTER_SEQUENCE) {
          result += `Message from ${
            msgInfo.chainId ? printChain(msgInfo.chainId) : ""
          }, with emitter address ${msgInfo.emitterAddress?.toString(
            "hex"
          )} and sequence number ${msgInfo.sequence}`;
        } else if (msgInfo.infoType == VaaKeyType.VAAHASH) {
          result += `VAA with hash ${msgInfo.vaaHash?.toString("hex")}`;
        } else {
          result += `VAA not specified correctly`;
        }
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
    stringifiedInfo += instruction.receiverValueTarget.gt(0)
      ? `Amount to pass into target address: ${instruction.receiverValueTarget} wei of ${targetChainName} currency\n`
      : ``;
    stringifiedInfo += `Gas limit: ${instruction.executionParameters.gasLimit} ${targetChainName} gas\n\n`;
    stringifiedInfo += info.targetChainStatus.events

            .map(
              (e, i) =>
                `Delivery attempt ${i + 1}: ${
                  e.transactionHash
                    ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
                    : ""
                }\nStatus: ${e.status}\n${e.revertString ? `Failure reason: ${e.gasUsed == instruction.executionParameters.gasLimit ? "Gas limit hit" : e.revertString}\n`: ""}Gas used: ${e.gasUsed}\nTransaction fee used: ${instruction.maximumRefundTarget.mul(e.gasUsed).div(instruction.executionParameters.gasLimit).toString()} wei of ${targetChainName} currency\n}`
            )
            .join("\n");
   } else if (info.type == RelayerPayloadId.Delivery && info.deliveryInstruction.targetAddress.toString("hex") === "0000000000000000000000000000000000000000000000000000000000000000") {
    stringifiedInfo += `Found delivery request in transaction ${
      info.sourceTransactionHash
    } on ${info.sourceChain}\n`;

    const instruction = info.deliveryInstruction;
    const targetChainName = CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];
    
    stringifiedInfo += `\nA refund of ${instruction.receiverValueTarget} ${targetChainName} wei was requested to be sent to ${targetChainName}, address 0x${info.deliveryInstruction.refundAddress.toString("hex")}`
    
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
  gasLimitForSendTransaction?: ethers.BigNumberish;
};

export async function send(
  sourceChain: ChainName,
  targetChain: ChainName,
  targetAddress: string,
  wallet: ethers.Wallet,
  payload: ethers.BytesLike,
  maxTransactionFee: ethers.BigNumberish,
  sendOptionalParams: SendOptionalParams
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
    wallet
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
  const tx = sourceCoreRelayer["send(uint16,bytes32,uint16,bytes32,uint256,uint256,bytes,(uint8,uint16,bytes32,uint64,bytes32)[],uint8,address,bytes)"](
    targetChainId, // targetChainId
    "0x" + tryNativeToHexString(targetAddress, "ethereum"), // targetAddress
    (refundLocationExists && sendOptionalParams.refundChainId) || sourceChainId, // refundChainId
    "0x" +
      tryNativeToHexString(
        (refundLocationExists &&
          sendOptionalParams.refundAddress &&
          sendOptionalParams.refundAddress) ||
          wallet.address,
        "ethereum"
      ), // refundAddress
    maxTransactionFee,
    sendOptionalParams?.receiverValue || 0, // receiverValue 
    payload,
    sendOptionalParams?.additionalVaas
      ? sendOptionalParams.additionalVaas.map(
          (additionalVaa): VaaKeyStruct => ({
            infoType: 0,
            chainId: additionalVaa.chainId || sourceChainId,
            emitterAddress: Buffer.from(tryNativeToHexString(
              additionalVaa.emitterAddress,
              "ethereum"
            ), "hex"),
            sequence: BigNumber.from(additionalVaa.sequenceNumber || 0),
            vaaHash: Buffer.from(""),
          })
        )
      : [], // vaaKeys
    sendOptionalParams?.consistencyLevel || 15, // consistencyLevel
    sendOptionalParams?.relayProviderAddress || defaultRelayProviderAddress, // relayProviderAddress
    sendOptionalParams?.relayParameters || Buffer.from(""), // relayParameters
  {
    value: maxTransactionFee,
    gasLimit: sendOptionalParams?.gasLimitForSendTransaction || 150000,
  });
  return tx;
}

export type GetPriceMultiHopOptParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  relayProviderAddress?: string;
  sourceChainProvider?: ethers.providers.Provider;
};

export type GetPriceOptParams = GetPriceMultiHopOptParams & {
  environment?: Network;
};

export async function getPrice(
  sourceChain: ChainName,
  targetChain: ChainName,
  gasAmount: ethers.BigNumberish,
  optionalParams?: GetPriceOptParams
): Promise<ethers.BigNumber> {
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
  const price = (
    await sourceCoreRelayer.quoteGas(
      targetChainId,
      gasAmount,
      relayProviderAddress
    )
  ).add(
    await sourceCoreRelayer.quoteReceiverValue(
      targetChainId,
      optionalParams?.receiverValue || 0,
      relayProviderAddress
    )
  );
  return price;
}

export type TargetChainAndGasAmount = {
  targetChain: ChainName,
  gasAmount: ethers.BigNumberish,
  optionalParams?: GetPriceMultiHopOptParams,
  forwards?: TargetChainAndGasAmount[]
}

export async function getPriceMultipleHops(sourceChain: ChainName, target: TargetChainAndGasAmount, environment: Network = "MAINNET"): Promise<ethers.BigNumber> {
  return _getPriceMultipleHops(sourceChain, [target], environment);
}

async function _getPriceMultipleHops(sourceChain: ChainName, targets: TargetChainAndGasAmount[], environment: Network = "MAINNET"): Promise<ethers.BigNumber> {
  let currentCost = BigNumber.from(0);
  for(let i = 0; i < targets.length; i++) {
    const optParams = { environment, ...targets[i].optionalParams };
    optParams.receiverValue = (targets[i].forwards ? await _getPriceMultipleHops(targets[i].targetChain, targets[i].forwards || [], environment) : BigNumber.from(0)).add(optParams.receiverValue || 0)
    currentCost = currentCost.add(await getPrice(sourceChain, targets[i].targetChain, targets[i].gasAmount, optParams));
  }
  return currentCost;
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
  newMaxTransactionFee: BigNumber | number,
  newReceiverValue: BigNumber | number,
  relayProviderAddress: string,
  overrides?: ethers.PayableOverrides
) {
  const provider = signer.provider;

  if (!provider) throw Error("No provider on signer");

  const coreRelayer = getWormholeRelayer(sourceChain, environment, signer);

  return coreRelayer.resend(
    vaaKeyToVaaKeyStruct(vaaKey),
    newMaxTransactionFee,
    newReceiverValue,
    CHAINS[targetChain],
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
  newMaxTransactionFee: BigNumber | number,
  newReceiverValue: BigNumber | number,
  relayProviderAddress: string,
  wormholeRPCs: string[],
  isNode?: boolean,
  overrides?: ethers.PayableOverrides
) {
  const sourceChainId = CHAINS[sourceChain];
  const targetChainId = CHAINS[targetChain];
  const originalVAA = await getVAA(wormholeRPCs, vaaKey, isNode);

  if (!originalVAA) throw Error("orignal VAA not found");

  const originalVAAparsed = parseWormholeRelayerSend(
    parseVaa(Buffer.from(originalVAA)).payload
  );
  if (!originalVAAparsed) throw Error("orignal VAA not a valid delivery VAA.");

  const originalGasLimit = originalVAAparsed.executionParameters.gasLimit;
  const originalMaxRefund = originalVAAparsed.maximumRefundTarget;
  const originalReceiverValue = originalVAAparsed.receiverValueTarget;
  const originalTargetChain = originalVAAparsed.targetChainId;

  if (originalTargetChain != targetChainId) {
    throw Error(
      `Target chain of original VAA (${originalTargetChain}) does not match target chain of resend (${targetChainId})`
    );
  }

  const coreRelayer = getWormholeRelayer(sourceChain, environment, signer);
  const relayProvider = getRelayProvider(
    relayProviderAddress,
    signer.provider!
  );

  const minimumReceiverValueSource = await coreRelayer.quoteReceiverValue(
    targetChainId,
    originalReceiverValue,
    relayProviderAddress
  );
  const minimumGasCoverageCost = await coreRelayer.quoteGas(
    targetChainId,
    originalGasLimit,
    relayProviderAddress
  );
  const minimumMaxRefundCost = await (
    await coreRelayer.quoteReceiverValue(
      targetChainId,
      originalMaxRefund,
      relayProviderAddress
    )
  ).add(await relayProvider.quoteDeliveryOverhead(targetChainId));

  const newMinimumMaxTransactionFee = minimumMaxRefundCost.gt(
    minimumGasCoverageCost
  )
    ? minimumMaxRefundCost
    : minimumGasCoverageCost;

  if (newMaxTransactionFee < newMinimumMaxTransactionFee) {
    throw Error(
      `New max transaction fee too low. Minimum is ${newMinimumMaxTransactionFee.toString()}`
    );
  }

  if (newReceiverValue < minimumReceiverValueSource) {
    throw Error(
      `New receiver value too low. Minimum is ${minimumReceiverValueSource.toString()}`
    );
  }

  return resendRaw(
    signer,
    sourceChain,
    targetChain,
    environment,
    vaaKey,
    newMaxTransactionFee,
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
  if (vaaKey.infoType != VaaKeyType.EMITTER_SEQUENCE) {
    throw Error("Hash vaa types not supported yet");
  }

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
