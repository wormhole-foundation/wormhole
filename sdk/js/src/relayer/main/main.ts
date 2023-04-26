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
import { getWormholeRelayerAddress } from "../consts";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  VaaKeyType,
  DeliveryStatus,
} from "../structs";
import {
  getDefaultProvider,
  printChain,
  getWormholeRelayerLog,
  parseWormholeLog,
  getBlockRange,
  getWormholeRelayerDeliveryEventsBySourceSequence,
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
    targetAddress: targetAddress,
    refundChain: (refundLocationExists && sendOptionalParams.refundChain) || sourceChain,
    refundAddress: (refundLocationExists && sendOptionalParams.refundAddress) || wallet.address,
    maxTransactionFee: maxTransactionFee,
    receiverValue: sendOptionalParams?.receiverValue || 0,
    relayProviderAddress: sendOptionalParams?.relayProviderAddress || defaultRelayProviderAddress,
    vaaKeys: sendOptionalParams?.additionalVaas ? sendOptionalParams.additionalVaas.map((additionalVaa): IWormholeRelayer.VaaKeyStruct => ({infoType: 0, chainId: additionalVaa.chainId || sourceChain, emitterAddress: tryHexToNativeString(additionalVaa.emitterAddress, "ethereum"), sequence: additionalVaa.sequenceNumber, vaaHash: ""})) : [],
    consistencyLevel: sendOptionalParams?.consistencyLevel || 15,
    payload: payload,
    relayParameters: sendOptionalParams?.relayParameters || ""
  }

  const tx = sourceCoreRelayer["send((uint16,bytes32,uint16,bytes32,uint256,uint256,address,(uint8,uint16,bytes32,uint64,bytes32)[],uint8,bytes,bytes))"](sendStruct, {value: maxTransactionFee, gasLimit: sendOptionalParams?.gasLimitForSendTransaction || 150000});
  return tx;
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
    type,
    sourceChainId: sourceChain,
    sourceTransactionHash: sourceTransaction,
    sourceDeliverySequenceNumber: BigNumber.from(
      deliveryLog.sequence
    ).toNumber(),
    deliveryInstruction: instruction,
    targetChainStatus,
  };
}
