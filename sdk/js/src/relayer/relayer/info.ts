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
  parseTransferPayload,
} from "../..";
import { BigNumber, ethers } from "ethers";
import { getWormholeRelayerAddress } from "../consts";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  RefundStatus,
  parseEVMExecutionInfoV1,
  DeliveryOverrideArgs,
  KeyType,
  parseVaaKey,
  parseCCTPKey,
  RedeliveryInstruction,
} from "../structs";
import {
  getDefaultProvider,
  printChain,
  printCCTPDomain,
  getWormholeLog,
  parseWormholeLog,
  getDeliveryHashFromLog,
  getRelayerTransactionHashFromWormscan,
  getWormholeRelayerInfoByHash,
  getWormscanRelayerInfo,
  getWormscanInfo,
  estimatedAttestationTimeInSeconds,
  getCCTPMessageLogURL,
} from "./helpers";
import {
  AdditionalMessageParsed,
  CCTPTransferParsed,
  DeliveryInfo,
  TokenTransferParsed,
} from "./deliver";
import { ERC20__factory } from "../../ethers-contracts";
import { IWormholeRelayer__factory } from "../../ethers-relayer-contracts";

export type InfoRequestParams = {
  environment?: Network;
  sourceChainProvider?: ethers.providers.Provider;
  targetChainProviders?: Map<ChainName, ethers.providers.Provider>;
  wormholeRelayerWhMessageIndex?: number;
  wormholeRelayerAddresses?: Map<ChainName, string>;
  targetBlockRange?: [ethers.providers.BlockTag, ethers.providers.BlockTag];
};

export type GetPriceOptParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  wormholeRelayerAddress?: string;
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
  const wormholeRelayerAddress =
    optionalParams?.wormholeRelayerAddress ||
    getWormholeRelayerAddress(sourceChain, environment);
  const sourceWormholeRelayer = IWormholeRelayer__factory.connect(
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
  if (!receipt)
    throw Error(
      `Transaction has not been mined: ${sourceTransaction} on ${sourceChain} (${environment})`
    );
  const sourceTimestamp =
    (await sourceChainProvider.getBlock(receipt.blockNumber)).timestamp * 1000;
  const bridgeAddress = CONTRACTS[environment][sourceChain].core;
  const wormholeRelayerAddress =
    infoRequest?.wormholeRelayerAddresses?.get(sourceChain) ||
    getWormholeRelayerAddress(sourceChain, environment);
  if (!bridgeAddress || !wormholeRelayerAddress) {
    throw Error(
      `Invalid chain ID or network: Chain ${sourceChain}, ${environment}`
    );
  }
  const deliveryLog = getWormholeLog(
    receipt,
    bridgeAddress,
    tryNativeToHexString(wormholeRelayerAddress, "ethereum"),
    infoRequest?.wormholeRelayerWhMessageIndex
      ? infoRequest.wormholeRelayerWhMessageIndex
      : 0
  );

  const { type, parsed } = parseWormholeLog(deliveryLog.log);

  if (type === RelayerPayloadId.Redelivery) {
    const redeliveryInstruction = parsed as RedeliveryInstruction;

    if (!isChain(redeliveryInstruction.deliveryVaaKey.chainId)) {
      throw new Error(
        `The chain ID specified by this redelivery is invalid: ${redeliveryInstruction.deliveryVaaKey.chainId}`
      );
    }
    if (!isChain(redeliveryInstruction.targetChainId)) {
      throw new Error(
        `The target chain ID specified by this redelivery is invalid: ${redeliveryInstruction.targetChainId}`
      );
    }

    const originalSourceChainName =
      CHAIN_ID_TO_NAME[redeliveryInstruction.deliveryVaaKey.chainId as ChainId];

    const modifiedInfoRequest = infoRequest;
    if (modifiedInfoRequest?.sourceChainProvider) {
      modifiedInfoRequest.sourceChainProvider =
        modifiedInfoRequest?.targetChainProviders?.get(originalSourceChainName);
    }

    const transactionHash = await getRelayerTransactionHashFromWormscan(
      originalSourceChainName,
      redeliveryInstruction.deliveryVaaKey.sequence.toNumber(),
      {
        network: infoRequest?.environment,
        provider: infoRequest?.targetChainProviders?.get(
          originalSourceChainName
        ),
        wormholeRelayerAddress: infoRequest?.wormholeRelayerAddresses?.get(
          originalSourceChainName
        ),
      }
    );

    return getWormholeRelayerInfo(
      originalSourceChainName,
      transactionHash,
      modifiedInfoRequest
    );
  }

  const instruction = parsed as DeliveryInstruction;

  const targetChainId = instruction.targetChainId;

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

  const sourceSequence = BigNumber.from(deliveryLog.sequence);

  const deliveryHash = await getDeliveryHashFromLog(
    deliveryLog.log,
    CHAINS[sourceChain],
    sourceChainProvider,
    receipt.blockHash
  );

  let signingOfVaaTimestamp;
  try {
    const vaa = await getWormscanRelayerInfo(
      sourceChain,
      sourceSequence.toNumber(),
      {
        network: infoRequest?.environment,
        provider: infoRequest?.sourceChainProvider,
        wormholeRelayerAddress:
          infoRequest?.wormholeRelayerAddresses?.get(sourceChain),
      }
    );
    signingOfVaaTimestamp = new Date(
      (await vaa.json()).data?.indexedAt
    ).getTime();
  } catch {
    // wormscan won't work for devnet - so let's hardcode this
    if (environment === "DEVNET") {
      signingOfVaaTimestamp = sourceTimestamp;
    }
  }

  // obtain additional message info
  const additionalMessageInformation: AdditionalMessageParsed[] =
    await Promise.all(
      instruction.messageKeys.map(async (messageKey) => {
        if (messageKey.keyType === 1) {
          // check receipt
          const vaaKey = parseVaaKey(messageKey.key);

          // if token bridge transfer in logs, parse it
          let tokenBridgeLog;
          const tokenBridgeEmitterAddress = tryNativeToHexString(
            CONTRACTS[environment][sourceChain].token_bridge || "",
            sourceChain
          );
          try {
            if (
              vaaKey.chainId === CHAINS[sourceChain] &&
              vaaKey.emitterAddress.toString("hex") ===
                tokenBridgeEmitterAddress
            ) {
              tokenBridgeLog = getWormholeLog(
                receipt,
                CONTRACTS[environment][sourceChain].core || "",
                tokenBridgeEmitterAddress,
                0,
                vaaKey.sequence.toNumber()
              );
            }
          } catch (e) {
            console.log(e);
          }
          if (!tokenBridgeLog) return undefined;
          const parsedTokenInfo = parseTransferPayload(
            Buffer.from(tokenBridgeLog.payload.substring(2), "hex")
          );
          const originChainName =
            CHAIN_ID_TO_NAME[parsedTokenInfo.originChain as ChainId];
          let signedVaaTimestamp = undefined;
          let tokenName = undefined;
          let tokenSymbol = undefined;
          let tokenDecimals = undefined;

          // Try to get additional token information, assuming it is an ERC20
          try {
            const tokenProvider =
              (parsedTokenInfo.originChain === CHAINS[sourceChain]
                ? infoRequest?.sourceChainProvider
                : infoRequest?.targetChainProviders?.get(originChainName)) ||
              getDefaultProvider(environment, originChainName);
            const tokenContract = ERC20__factory.connect(
              "0x" + parsedTokenInfo.originAddress.substring(24),
              tokenProvider
            );
            tokenName = await tokenContract.name();
            tokenSymbol = await tokenContract.symbol();
            tokenDecimals = await tokenContract.decimals();
          } catch (e) {
            console.log(e);
          }
          // Try to get wormscan information on if the tokens have been signed
          try {
            const tokenVaa = await getWormscanInfo(
              environment,
              sourceChain,
              parseInt(tokenBridgeLog.sequence),
              CONTRACTS[environment][sourceChain].token_bridge || ""
            );
            signedVaaTimestamp = new Date(
              (await tokenVaa.json()).data?.indexedAt
            ).getTime();
          } catch {}

          const parsed: TokenTransferParsed = {
            amount: BigNumber.from(parsedTokenInfo.amount)
              .mul(
                BigNumber.from(10).pow(
                  tokenDecimals && tokenDecimals > 8 ? tokenDecimals - 8 : 1
                )
              )
              .toBigInt(),
            originAddress: parsedTokenInfo.originAddress,
            originChain: parsedTokenInfo.originChain,
            targetAddress: parsedTokenInfo.targetAddress,
            targetChain: parsedTokenInfo.targetChain,
            fromAddress: parsedTokenInfo.fromAddress,
            name: tokenName,
            symbol: tokenSymbol,
            decimals: tokenDecimals,
            signedVaaTimestamp,
          };
          return parsed;
        } else if (messageKey.keyType === 2) {
          // check receipt
          const cctpKey = parseCCTPKey(messageKey.key);
          const cctpInfo = await getCCTPMessageLogURL(
            cctpKey,
            sourceChain,
            receipt,
            environment
          );
          const url = cctpInfo?.url || "";

          // Try to get attestation information on if the tokens have been signed
          let attested = false;
          try {
            const attestation = await fetch(url);
            attested = (await attestation.json()).status === "complete";
          } catch (e) {
            console.log(e);
          }
          const cctpLog = cctpInfo?.cctpLog!;
          const parsed: CCTPTransferParsed = {
            amount: BigNumber.from(
              Buffer.from(cctpLog.data.substring(2, 2 + 64), "hex")
            ).toBigInt(),
            mintRecipient: "0x" + cctpLog.data.substring(2 + 64 + 24, 2 + 128),
            destinationDomain: BigNumber.from(
              Buffer.from(cctpLog.data.substring(2 + 128, 2 + 192), "hex")
            ).toNumber(),
            attested,
            estimatedAttestationSeconds: estimatedAttestationTimeInSeconds(
              sourceChain,
              environment
            ),
          };
          return parsed;
        } else {
          return undefined;
        }
      })
    );

  const targetChainDeliveries = await getWormholeRelayerInfoByHash(
    deliveryHash,
    targetChain,
    sourceChain,
    sourceSequence.toNumber(),
    infoRequest
  );

  const result: DeliveryInfo = {
    type: RelayerPayloadId.Delivery,
    sourceChain: sourceChain,
    sourceTransactionHash: sourceTransaction,
    sourceDeliverySequenceNumber: sourceSequence.toNumber(),
    deliveryInstruction: instruction,
    sourceTimestamp,
    signingOfVaaTimestamp,
    additionalMessageInformation,
    targetChainStatus: {
      chain: targetChain,
      events: targetChainDeliveries,
    },
  };
  const stringified = stringifyWormholeRelayerInfo(result);
  result.stringified = stringified;
  return result;
}

export function printWormholeRelayerInfo(info: DeliveryInfo) {
  console.log(stringifyWormholeRelayerInfo(info));
}

export function stringifyWormholeRelayerInfo(
  info: DeliveryInfo,
  excludeSourceInformation?: boolean,
  overrides?: DeliveryOverrideArgs
): string {
  let stringifiedInfo = "";
  if (
    info.type == RelayerPayloadId.Delivery &&
    info.deliveryInstruction.targetAddress.toString("hex") !==
      "0000000000000000000000000000000000000000000000000000000000000000"
  ) {
    if (!excludeSourceInformation) {
      stringifiedInfo += `Source chain: ${info.sourceChain}\n`;

      stringifiedInfo += `Source Transaction Hash: ${info.sourceTransactionHash}\n`;
      stringifiedInfo += `Sender: ${
        "0x" +
        info.deliveryInstruction.senderAddress.toString("hex").substring(24)
      }\n`;
      stringifiedInfo += `Delivery sequence number: ${info.sourceDeliverySequenceNumber}\n`;
    } else {
      stringifiedInfo += `Sender: ${info.deliveryInstruction.senderAddress.toString(
        "hex"
      )}\n`;
    }
    const numMsgs = info.deliveryInstruction.messageKeys.length;

    const payload = info.deliveryInstruction.payload.toString("hex");
    if (payload.length > 0) {
      stringifiedInfo += `\nPayload to be relayed: 0x${payload}\n`;
    }
    if (numMsgs > 0) {
      stringifiedInfo += `\nThe following ${
        numMsgs === 1 ? "" : `${numMsgs} `
      }message${numMsgs === 1 ? " was" : "s were"} ${
        payload.length > 0 ? "also " : ""
      }requested to be relayed with this delivery:\n`;
      stringifiedInfo += info.deliveryInstruction.messageKeys
        .map((msgKey, i) => {
          let result = "";
          if (msgKey.keyType == KeyType.VAA) {
            const vaaKey = parseVaaKey(msgKey.key);
            result += `(Message ${i + 1}): `;
            result += `Wormhole VAA from ${
              vaaKey.chainId ? printChain(vaaKey.chainId) : ""
            }, with emitter address ${vaaKey.emitterAddress?.toString(
              "hex"
            )} and sequence number ${vaaKey.sequence}`;
            if (info.additionalMessageInformation[i]) {
              const tokenTransferInfo = info.additionalMessageInformation[
                i
              ] as TokenTransferParsed;
              result += `\nThis is a token bridge transfer of ${
                tokenTransferInfo.decimals
                  ? `${ethers.utils.formatUnits(
                      tokenTransferInfo.amount,
                      tokenTransferInfo.decimals
                    )} `
                  : `${tokenTransferInfo.amount} normalized units of `
              }${
                tokenTransferInfo.name
                  ? `${tokenTransferInfo.name} (${tokenTransferInfo.symbol})`
                  : `token ${tokenTransferInfo.originAddress.substring(
                      24
                    )} (which is native to ${printChain(
                      tokenTransferInfo.originChain
                    )})`
              }`;
              if (tokenTransferInfo.signedVaaTimestamp) {
                result += `\ntransfer signed by guardians: ${new Date(
                  tokenTransferInfo.signedVaaTimestamp
                ).toString()}`;
              } else {
                result += `\ntransfer not yet signed by guardians`;
              }
            }
          } else if (msgKey.keyType == KeyType.CCTP) {
            const cctpKey = parseCCTPKey(msgKey.key);
            result += `(Message ${i + 1}): `;
            result += `CCTP Transfer from domain ${printCCTPDomain(
              cctpKey.domain
            )}`;
            result += `, with nonce ${cctpKey.nonce}`;
            if (info.additionalMessageInformation[i]) {
              const cctpTransferInfo = info.additionalMessageInformation[
                i
              ] as CCTPTransferParsed;
              result += `\nThis is a CCTP transfer of ${`${ethers.utils.formatUnits(
                cctpTransferInfo.amount,
                6
              )}`} USDC ${
                cctpTransferInfo.attested
                  ? "(Attestation is complete"
                  : "(Attestation currently pending"
              }, typically takes ${
                cctpTransferInfo.estimatedAttestationSeconds < 60
                  ? `${cctpTransferInfo.estimatedAttestationSeconds} seconds`
                  : `${
                      cctpTransferInfo.estimatedAttestationSeconds / 60
                    } minutes`
              })`;
            }
          } else {
            result += `(Unknown key type ${i}): ${msgKey.keyType}`;
          }
          return result;
        })
        .join(",\n");
    }
    if (payload.length == 0 && numMsgs == 0) {
      stringifiedInfo += `\nAn empty payload was requested to be sent`;
    }

    const instruction = info.deliveryInstruction;
    if (overrides) {
      instruction.requestedReceiverValue = overrides.newReceiverValue;
      instruction.encodedExecutionInfo = overrides.newExecutionInfo;
    }
    const targetChainName =
      CHAIN_ID_TO_NAME[instruction.targetChainId as ChainId];
    stringifiedInfo += `\n\nDestination chain: ${printChain(
      instruction.targetChainId
    )}\nDestination address: 0x${instruction.targetAddress
      .toString("hex")
      .substring(24)}\n\n`;
    const totalReceiverValue = instruction.requestedReceiverValue.add(
      instruction.extraReceiverValue
    );
    stringifiedInfo += totalReceiverValue.gt(0)
      ? `Amount to pass into target address: ${ethers.utils.formatEther(
          totalReceiverValue
        )} of ${targetChainName} currency ${
          instruction.extraReceiverValue.gt(0)
            ? `\n${ethers.utils.formatEther(
                instruction.requestedReceiverValue
              )} requested, ${ethers.utils.formatEther(
                instruction.extraReceiverValue
              )} additionally paid for`
            : ""
        }\n`
      : ``;
    const [executionInfo] = parseEVMExecutionInfoV1(
      instruction.encodedExecutionInfo,
      0
    );
    stringifiedInfo += `Gas limit: ${executionInfo.gasLimit} ${targetChainName} gas\n`;

    const refundAddressChosen =
      instruction.refundAddress.toString("hex") !==
      "0000000000000000000000000000000000000000000000000000000000000000";
    if (refundAddressChosen) {
      stringifiedInfo += `Refund rate: ${ethers.utils.formatEther(
        executionInfo.targetChainRefundPerGasUnused
      )} of ${targetChainName} currency per unit of gas unused\n`;
      stringifiedInfo += `Refund address: ${instruction.refundAddress.toString(
        "hex"
      )} on ${printChain(instruction.refundChainId)}\n`;
    }
    stringifiedInfo += `\n`;
    if (info.sourceTimestamp) {
      stringifiedInfo += `Sent: ${new Date(info.sourceTimestamp).toString()}\n`;
    }
    if (info.signingOfVaaTimestamp) {
      stringifiedInfo += `Delivery vaa signed by guardians: ${new Date(
        info.signingOfVaaTimestamp
      ).toString()}\n`;
    } else {
      stringifiedInfo += `Delivery not yet signed by guardians - check https://wormhole-foundation.github.io/wormhole-dashboard/#/ for status\n`;
    }
    stringifiedInfo += `\n`;
    if (info.targetChainStatus.events.length === 0) {
      stringifiedInfo += "Delivery has not occured yet\n";
    }
    stringifiedInfo += info.targetChainStatus.events

      .map((e, i) => {
        let override = e.overrides || false;
        let overriddenExecutionInfo = e.overrides
          ? parseEVMExecutionInfoV1(e.overrides.newExecutionInfo, 0)[0]
          : executionInfo;
        let overriddenReceiverValue = e.overrides
          ? e.overrides.newReceiverValue
          : totalReceiverValue;
        const overriddenGasLimit = override
          ? overriddenExecutionInfo.gasLimit
          : executionInfo.gasLimit;

        // Add information about any override applied to the delivery
        let overrideStringifiedInfo = "";
        if (override) {
          overrideStringifiedInfo += !overriddenReceiverValue.eq(
            totalReceiverValue
          )
            ? `Overridden amount to pass into target address: ${ethers.utils.formatEther(
                overriddenReceiverValue
              )} of ${targetChainName} currency\n`
            : ``;
          overrideStringifiedInfo += !(
            overriddenGasLimit === executionInfo.gasLimit
          )
            ? `Overridden gas limit: ${overriddenExecutionInfo.gasLimit} ${targetChainName} gas\n`
            : "";
          if (
            refundAddressChosen &&
            executionInfo.targetChainRefundPerGasUnused !==
              overriddenExecutionInfo.targetChainRefundPerGasUnused
          ) {
            overrideStringifiedInfo += `Overridden refund rate: ${ethers.utils.formatEther(
              overriddenExecutionInfo.targetChainRefundPerGasUnused
            )} of ${targetChainName} currency per unit of gas unused\n`;
          }
        }

        return `Delivery attempt: ${
          e.transactionHash
            ? ` ${targetChainName} transaction hash: ${e.transactionHash}`
            : ""
        }\nDelivery Time: ${new Date(
          e.timestamp as number
        ).toString()}\n${overrideStringifiedInfo}Status: ${e.status}\n${
          e.revertString
            ? `Failure reason: ${
                e.gasUsed.eq(overriddenExecutionInfo.gasLimit)
                  ? "Gas limit hit"
                  : e.revertString
              }\n`
            : ""
        }Gas used: ${e.gasUsed.toString()}\nTransaction fee used: ${ethers.utils.formatEther(
          overriddenExecutionInfo.targetChainRefundPerGasUnused.mul(e.gasUsed)
        )} of ${targetChainName} currency\n${`Refund amount: ${ethers.utils.formatEther(
          overriddenExecutionInfo.targetChainRefundPerGasUnused.mul(
            overriddenExecutionInfo.gasLimit.sub(e.gasUsed)
          )
        )} of ${targetChainName} currency \nRefund status: ${
          e.refundStatus
        }\n`}`;
      })
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

    stringifiedInfo += `\nA refund of ${ethers.utils.formatEther(
      instruction.extraReceiverValue
    )} ${targetChainName} currency was requested to be sent to ${targetChainName}, address 0x${info.deliveryInstruction.refundAddress.toString(
      "hex"
    )}\n\n`;

    stringifiedInfo += info.targetChainStatus.events

      .map(
        (e, i) =>
          `Delivery attempt: ${
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
