import {
  ChainId,
  CHAIN_ID_TO_NAME,
  ChainName,
  isChain,
  CONTRACTS,
  CHAINS,
  tryNativeToHexString,
  Network,
  ethers_contracts,
} from "../..";
import { BigNumber, ethers } from "ethers";
import { getWormholeRelayerAddress } from "../consts";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  RefundStatus,
  parseEVMExecutionInfoV1,
} from "../structs";
import {
  getDefaultProvider,
  printChain,
  getWormholeRelayerLog,
  parseWormholeLog,
  getBlockRange,
  getWormholeRelayerInfoBySourceSequence,
} from "./helpers";
import { DeliveryInfo } from "./deliver";

export type InfoRequestParams = {
  environment?: Network;
  sourceChainProvider?: ethers.providers.Provider;
  targetChainProviders?: Map<ChainName, ethers.providers.Provider>;
  targetChainBlockRanges?: Map<
    ChainName,
    [ethers.providers.BlockTag, ethers.providers.BlockTag]
  >;
  wormholeRelayerWhMessageIndex?: number;
  wormholeRelayerAddresses?: Map<ChainName, string>;
};


export type GetPriceOptParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  deliveryProviderAddress?: string;
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
  const wormholeRelayerAddress = getWormholeRelayerAddress(
    sourceChain,
    environment
  );
  console.log(`Wormhole relayer address ${wormholeRelayerAddress}`);
  console.log(`Provider ${sourceChainProvider}`);
  const sourceWormholeRelayer =
    ethers_contracts.IWormholeRelayer__factory.connect(
      wormholeRelayerAddress,
      sourceChainProvider
    );
  const deliveryProviderAddress =
    optionalParams?.deliveryProviderAddress ||
    (await sourceWormholeRelayer.getDefaultDeliveryProvider());
  const targetChainId = CHAINS[targetChain];
  const priceAndRefundInfo = await sourceWormholeRelayer[
    "quoteEVMDeliveryPrice(uint16,uint256,uint256,address)"
  ](
    targetChainId,
    optionalParams?.receiverValue || 0,
    gasAmount,
    deliveryProviderAddress
  );
  return priceAndRefundInfo;
}

export async function getPrice(
  sourceChain: ChainName,
  targetChain: ChainName,
  gasAmount: ethers.BigNumberish,
  optionalParams?: GetPriceOptParams
): Promise<ethers.BigNumber> {
  const priceAndRefundInfo = await getPriceAndRefundInfo(
    sourceChain,
    targetChain,
    gasAmount,
    optionalParams
  );
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
  const bridgeAddress = CONTRACTS[environment][sourceChain].core;
  const wormholeRelayerAddress =
    infoRequest?.wormholeRelayerAddresses?.get(sourceChain) ||
    getWormholeRelayerAddress(sourceChain, environment);
  if (!bridgeAddress || !wormholeRelayerAddress) {
    throw Error(
      `Invalid chain ID or network: Chain ${sourceChain}, ${environment}`
    );
  }
  const deliveryLog = getWormholeRelayerLog(
    receipt,
    bridgeAddress,
    tryNativeToHexString(wormholeRelayerAddress, "ethereum"),
    infoRequest?.wormholeRelayerWhMessageIndex
      ? infoRequest.wormholeRelayerWhMessageIndex
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
    infoRequest?.wormholeRelayerAddresses?.get(targetChain) ||
      getWormholeRelayerAddress(targetChain, environment)
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

export function printWormholeRelayerInfo(info: DeliveryInfo) {
  console.log(stringifyWormholeRelayerInfo(info));
}

export function stringifyWormholeRelayerInfo(info: DeliveryInfo): string {
  let stringifiedInfo = "";
  if (
    info.type == RelayerPayloadId.Delivery &&
    info.deliveryInstruction.targetAddress.toString("hex") !==
      "0000000000000000000000000000000000000000000000000000000000000000"
  ) {
    stringifiedInfo += `Found delivery request in transaction ${info.sourceTransactionHash} on ${info.sourceChain}\n`;
    const numMsgs = info.deliveryInstruction.vaaKeys.length;

    const payload = info.deliveryInstruction.payload.toString("hex");
    if (payload.length > 0) {
      stringifiedInfo += `\nPayload to be relayed (as hex string): 0x${payload}`;
    }
    if (numMsgs > 0) {
      stringifiedInfo += `\nThe following ${numMsgs} wormhole messages (VAAs) were ${
        payload.length > 0 ? "also " : ""
      }requested to be relayed:\n`;
      stringifiedInfo += info.deliveryInstruction.vaaKeys
        .map((msgInfo, i) => {
          let result = "";
          result += `(VAA ${i}): `;
          result += `Message from ${
            msgInfo.chainId ? printChain(msgInfo.chainId) : ""
          }, with emitter address ${msgInfo.emitterAddress?.toString(
            "hex"
          )} and sequence number ${msgInfo.sequence}`;

          return result;
        })
        .join(",\n");
    }
    if (payload.length == 0 && numMsgs == 0) {
      stringifiedInfo += `\nAn empty payload was requested to be sent`;
    }

    const instruction = info.deliveryInstruction;
    const targetChainName =
      CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];
    stringifiedInfo += `${
      numMsgs == 0
        ? payload.length == 0
          ? ""
          : "\n\nPayload was requested to be relayed"
        : "\n\nThese were requested to be sent"
    } to 0x${instruction.targetAddress.toString("hex")} on ${printChain(
      instruction.targetChainId
    )}\n`;
    const totalReceiverValue = instruction.requestedReceiverValue.add(
      instruction.extraReceiverValue
    );
    stringifiedInfo += totalReceiverValue.gt(0)
      ? `Amount to pass into target address: ${ethers.utils.formatEther(totalReceiverValue)} of ${targetChainName} currency ${
          instruction.extraReceiverValue.gt(0)
            ? `\n${ethers.utils.formatEther(instruction.requestedReceiverValue)} requested, ${ethers.utils.formatEther(instruction.extraReceiverValue)} additionally paid for`
            : ""
        }\n`
      : ``;
    const [executionInfo] = parseEVMExecutionInfoV1(
      instruction.encodedExecutionInfo,
      0
    );
    stringifiedInfo += `Gas limit: ${executionInfo.gasLimit} ${targetChainName} gas\n`;

    const refundAddressChosen = instruction.refundAddress !== instruction.refundDeliveryProvider;
    if(refundAddressChosen) {
      stringifiedInfo += `Refund rate: ${ethers.utils.formatEther(executionInfo.targetChainRefundPerGasUnused)} of ${targetChainName} currency per unit of gas unused\n`;
      stringifiedInfo += `Refund address: ${instruction.refundAddress.toString("hex")}\n`
    }
    stringifiedInfo += `\n`
    stringifiedInfo += info.targetChainStatus.events

      .map(
        (e, i) =>
          `Delivery attempt ${i + 1}: ${
            e.transactionHash
              ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
              : ""
          }\nStatus: ${e.status}\n${
            e.revertString
              ? `Failure reason: ${
                  e.gasUsed.eq(executionInfo.gasLimit)
                    ? "Gas limit hit"
                    : e.revertString
                }\n`
              : ""
          }Gas used: ${e.gasUsed.toString()}\nTransaction fee used: ${ethers.utils.formatEther(executionInfo.targetChainRefundPerGasUnused
            .mul(e.gasUsed))} of ${targetChainName} currency\n${(!refundAddressChosen || e.status === "Forward Request Success") ? "" : `Refund amount: ${ethers.utils.formatEther(executionInfo.targetChainRefundPerGasUnused.mul(executionInfo.gasLimit.sub(e.gasUsed)))} of ${targetChainName} currency \nRefund status: ${e.refundStatus}\n`}`
      )
      .join("\n");
  } else if (
    info.type == RelayerPayloadId.Delivery &&
    info.deliveryInstruction.targetAddress.toString("hex") ===
      "0000000000000000000000000000000000000000000000000000000000000000"
  ) {
    stringifiedInfo += `Found delivery request in transaction ${info.sourceTransactionHash} on ${info.sourceChain}\n`;

    const instruction = info.deliveryInstruction;
    const targetChainName =
      CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];

    stringifiedInfo += `\nA refund of ${
      ethers.utils.formatEther(instruction.extraReceiverValue)
    } ${targetChainName} currency was requested to be sent to ${targetChainName}, address 0x${info.deliveryInstruction.refundAddress.toString(
      "hex"
    )}`;

    stringifiedInfo += info.targetChainStatus.events

      .map(
        (e, i) =>
          `Delivery attempt ${i + 1}: ${
            e.transactionHash
              ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
              : ""
          }\nStatus: ${
            e.refundStatus == RefundStatus.RefundSent
              ? "Refund Successful"
              : "Refund Failed"
          }`
      )
      .join("\n");
  }

  return stringifiedInfo;
}
