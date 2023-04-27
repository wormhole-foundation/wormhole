import {
  ChainId,
  CHAIN_ID_TO_NAME,
  isChain,
  CONTRACTS,
  tryNativeToHexString,
  tryHexToNativeString,
  Network,
  ethers_contracts
} from "../../";
import { BigNumber, ethers } from "ethers";
import { getWormholeRelayer, getWormholeRelayerAddress } from "../consts";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  VaaKeyType,
  DeliveryStatus,
  VaaKey,
} from "../structs";
import {
  getDefaultProvider,
  printChain,
  getWormholeRelayerLog,
  parseWormholeLog,
  getBlockRange,
  getWormholeRelayerDeliveryEventsBySourceSequence,
  vaaKeyToVaaKeyStruct,
} from "./helpers";
import { IWormholeRelayer } from "../../ethers-contracts";

export type InfoRequestParams = {
  environment?: Network;
  sourceChainProvider?: ethers.providers.Provider;
  targetChainProviders?: Map<number, ethers.providers.Provider>;
  targetChainBlockRanges?: Map<
    number,
    [ethers.providers.BlockTag, ethers.providers.BlockTag]
  >;
  coreRelayerWhMessageIndex?: number;
};

export type DeliveryInfo = {
  type: RelayerPayloadId.Delivery;
  sourceChainId: ChainId;
  sourceTransactionHash: string;
  sourceDeliverySequenceNumber: number;
  deliveryInstruction: DeliveryInstruction;
  targetChainStatus: {
    chainId: ChainId;
    events: {
      status: DeliveryStatus | string;
      transactionHash: string | null;
    }[];
  };
};

export function printWormholeRelayerInfo(info: DeliveryInfo) {
  console.log(stringifyWormholeRelayerInfo(info));
}

export function stringifyWormholeRelayerInfo(info: DeliveryInfo): string {
  let stringifiedInfo = "";
  if (info.type == RelayerPayloadId.Delivery) {
    stringifiedInfo += `Found delivery request in transaction ${
      info.sourceTransactionHash
    } on ${printChain(info.sourceChainId)}\n`;

    const numMsgs = info.deliveryInstruction.vaaKeys.length;
    stringifiedInfo += `\nThe following ${numMsgs} wormhole messages (VAAs) were requested to be relayed:\n`;
    stringifiedInfo += info.deliveryInstruction.vaaKeys.map((msgInfo, i) => {
      let result = "";
      result += `(VAA ${i}): `;
      if (msgInfo.payloadType == VaaKeyType.EMITTER_SEQUENCE) {
        result += `Message from ${
          msgInfo.chainId ? printChain(msgInfo.chainId) : ""
        }, with emitter address ${msgInfo.emitterAddress?.toString(
          "hex"
        )} and sequence number ${msgInfo.sequence}`;
      } else if (msgInfo.payloadType == VaaKeyType.VAAHASH) {
        result += `VAA with hash ${msgInfo.vaaHash?.toString("hex")}`;
      } else {
        result += `VAA not specified correctly`;
      }
      return result;
    }).join(",\n");

    const length = 1;
    const instruction = info.deliveryInstruction;
    const targetChainName = CHAIN_ID_TO_NAME[instruction.targetChain as ChainId];
    stringifiedInfo += `\n\nVAAs were requested to be sent to 0x${instruction.targetAddress.toString(
      "hex"
    )} on ${printChain(instruction.targetChain)}\n`;
    stringifiedInfo += instruction.receiverValueTarget.gt(0)
            ? `Amount to pass into target address: ${
                instruction.receiverValueTarget
              } wei of ${
                targetChainName
              } currency\n`
            : ``;
    stringifiedInfo += `Gas limit: ${instruction.executionParameters.gasLimit} ${targetChainName} gas\n`;
    stringifiedInfo += info.targetChainStatus.events
            .map(
              (e, i) =>
                `Delivery attempt ${i + 1}: ${e.status}${
                  e.transactionHash
                    ? ` (${targetChainName} transaction hash: ${e.transactionHash})`
                    : ""
                }`
            )
            .join("\n");
  return stringifiedInfo;
              }
}

export type SendOptionalParams = {
  environment?: Network,
  receiverValue?: ethers.BigNumberish,
  additionalVaas?: [{chainId?: ChainId, emitterAddress: string, sequenceNumber: ethers.BigNumberish}],
  relayProviderAddress?: string,
  consistencyLevel?: ethers.BigNumberish,
  refundChain?: ChainId,
  refundAddress?: string
  relayParameters?: ethers.BytesLike,
  gasLimitForSendTransaction?: ethers.BigNumberish
}

export async function send(sourceChain: ChainId, targetChain: ChainId, targetAddress: string, wallet: ethers.Wallet, payload: ethers.BytesLike, maxTransactionFee: ethers.BigNumberish, sendOptionalParams: SendOptionalParams): Promise<ethers.providers.TransactionResponse> {
  const environment = sendOptionalParams?.environment || "MAINNET";
  const coreRelayerAddress = getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  const sourceCoreRelayer = ethers_contracts.IWormholeRelayer__factory.connect(
    coreRelayerAddress,
    wallet
  );

  const refundLocationExists = (sendOptionalParams?.refundChain !== undefined && sendOptionalParams?.refundAddress !== undefined);
  const defaultRelayProviderAddress = await sourceCoreRelayer.getDefaultRelayProvider();
  const sendStruct: IWormholeRelayer.SendStruct = {
    targetChain: targetChain,
    targetAddress: "0x"+tryNativeToHexString(targetAddress, "ethereum"),
    refundChain: (refundLocationExists && sendOptionalParams.refundChain) || sourceChain,
    refundAddress: "0x"+tryNativeToHexString((refundLocationExists && sendOptionalParams.refundAddress && sendOptionalParams.refundAddress) || wallet.address,"ethereum"),
    maxTransactionFee: maxTransactionFee,
    receiverValue: sendOptionalParams?.receiverValue || 0,
    relayProviderAddress: sendOptionalParams?.relayProviderAddress || defaultRelayProviderAddress,
    vaaKeys: sendOptionalParams?.additionalVaas ? sendOptionalParams.additionalVaas.map((additionalVaa): IWormholeRelayer.VaaKeyStruct => ({infoType: 0, chainId: additionalVaa.chainId || sourceChain, emitterAddress: tryNativeToHexString(additionalVaa.emitterAddress, "ethereum"), sequence: additionalVaa.sequenceNumber, vaaHash: ""})) : [],
    consistencyLevel: sendOptionalParams?.consistencyLevel || 15,
    payload: payload,
    relayParameters: sendOptionalParams?.relayParameters || Buffer.from("")
  }

  const tx = sourceCoreRelayer["send((uint16,bytes32,uint16,bytes32,uint256,uint256,address,(uint8,uint16,bytes32,uint64,bytes32)[],uint8,bytes,bytes))"](sendStruct, {value: maxTransactionFee, gasLimit: sendOptionalParams?.gasLimitForSendTransaction || 150000});
  return tx;
}

export type GetPriceMultiHopOptParams = {
  environment?: Network,
  receiverValue?: ethers.BigNumberish,
  relayProviderAddress?: string,
  sourceChainProvider?: ethers.providers.Provider 
}

export type GetPriceOptParams = GetPriceMultiHopOptParams & {
  environment?: Network
}

export async function getPrice(sourceChain: ChainId, targetChain: ChainId, gasAmount: ethers.BigNumberish, optionalParams?: GetPriceOptParams): Promise<ethers.BigNumber> {
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
  const relayProviderAddress = optionalParams?.relayProviderAddress || (await sourceCoreRelayer.getDefaultRelayProvider());
  const price = (await sourceCoreRelayer.quoteGas(targetChain, gasAmount, relayProviderAddress)).add(await sourceCoreRelayer.quoteReceiverValue(targetChain, optionalParams?.receiverValue || 0, relayProviderAddress));
  return price
}

export async function getPriceMultipleHops(sourceChain: ChainId, targets: {targetChain: ChainId, gasAmount: ethers.BigNumberish, optionalParams?: GetPriceMultiHopOptParams}[], environment: Network = "MAINNET"): ethers.BigNumber {
  const chains = [sourceChain].concat(targets.map((t)=>t.targetChain));
  let currentCost = BigNumber.from(0);
  for(let i=chains.length-2; i>=0; i--) {
    const optParams = {environment, ...targets[i].optionalParams};
    optParams.receiverValue = currentCost.add(optParams.receiverValue || 0);
    currentCost = await getPrice(chains[i], chains[i+1], targets[i].gasAmount, optParams);
  }
  return currentCost;
}

export async function getWormholeRelayerInfo(
  sourceChain: ChainId, sourceTransaction: string, infoRequest?: InfoRequestParams
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
    CONTRACTS[environment][
      CHAIN_ID_TO_NAME[sourceChain]
    ].core;
  const coreRelayerAddress = getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  if (!bridgeAddress || !coreRelayerAddress) {
    throw Error(
      `Invalid chain ID or network: Chain ID ${sourceChain}, ${environment}`
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

  const targetChain = instruction.targetChain as ChainId;
  if (!isChain(targetChain)) throw Error(`Invalid Chain: ${targetChain}`);
  const targetChainProvider =
    infoRequest?.targetChainProviders?.get(targetChain) ||
    getDefaultProvider(environment, targetChain);

  if (!targetChainProvider)
    throw Error(
      "No default RPC for this chain; pass in your own provider (as targetChainProvider)"
    );

  const sourceChainBlock = await sourceChainProvider.getBlock(
    receipt.blockNumber
  );
  const [blockStartNumber, blockEndNumber] =
    infoRequest?.targetChainBlockRanges?.get(targetChain) ||
    getBlockRange(targetChainProvider, sourceChainBlock.timestamp);

  const deliveryEvents = await getWormholeRelayerDeliveryEventsBySourceSequence(
    environment,
    targetChain,
    targetChainProvider,
    sourceChain,
    BigNumber.from(deliveryLog.sequence),
    blockStartNumber,
    blockEndNumber
  );
  if (deliveryEvents.length == 0) {
    let status = `Delivery didn't happen on ${printChain(
      targetChain
    )} within blocks ${blockStartNumber} to ${blockEndNumber}.`;
    try {
      const blockStart = await targetChainProvider.getBlock(blockStartNumber);
      const blockEnd = await targetChainProvider.getBlock(blockEndNumber);
      status = `Delivery didn't happen on ${printChain(
        targetChain
      )} within blocks ${blockStart.number} to ${
        blockEnd.number
      } (within times ${new Date(
        blockStart.timestamp * 1000
      ).toString()} to ${new Date(blockEnd.timestamp * 1000).toString()})`;
    } catch (e) {}
    deliveryEvents.push({
      status,
      deliveryTxHash: null,
      vaaHash: null,
      sourceChain: sourceChain,
      sourceVaaSequence: BigNumber.from(deliveryLog.sequence),
    });
  }
  const targetChainStatus = {
    chainId: targetChain,
    events: deliveryEvents.map((e) => ({
      status: e.status,
      transactionHash: e.deliveryTxHash,
    })),
  };

  return {
    type: RelayerPayloadId.Delivery,
    sourceChainId: sourceChain,
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
  sourceChain: ChainId,
  targetChain: ChainId,
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
    targetChain,
    relayProviderAddress,
    overrides
  );
}
