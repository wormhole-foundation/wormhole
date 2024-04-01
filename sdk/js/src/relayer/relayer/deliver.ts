import { BigNumber, ethers, ContractReceipt } from "ethers";
import { IWormholeRelayer__factory } from "../../ethers-relayer-contracts";
import {
  ChainName,
  toChainName,
  ChainId,
  Network,
  CHAIN_ID_TO_NAME,
} from "../../utils";
import { SignedVaa, parseVaa } from "../../vaa";
import { getWormholeRelayerAddress } from "../consts";
import {
  RelayerPayloadId,
  DeliveryInstruction,
  DeliveryOverrideArgs,
  packOverrides,
  parseEVMExecutionInfoV1,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
  parseVaaKey,
  MessageKey,
  parseCCTPKey,
} from "../structs";
import {
  DeliveryTargetInfo,
  getCCTPMessageLogURL,
  getDefaultProvider,
  getWormscanInfo,
} from "./helpers";
import { InfoRequestParams, getWormholeRelayerInfo } from "./info";

export type CCTPTransferParsed = {
  amount: bigint; // decimals is 6
  mintRecipient: string;
  destinationDomain: number;
  estimatedAttestationSeconds: number;
  attested: boolean;
};
export type TokenTransferParsed = {
  amount: bigint;
  originAddress: string;
  originChain: number;
  targetAddress: string;
  targetChain: number;
  fromAddress: string | undefined;
  name?: string;
  symbol?: string;
  decimals?: number;
  signedVaaTimestamp?: number;
};
export type AdditionalMessageParsed =
  | CCTPTransferParsed
  | TokenTransferParsed
  | undefined;

export type DeliveryInfo = {
  type: RelayerPayloadId.Delivery;
  sourceChain: ChainName;
  sourceTransactionHash: string;
  sourceDeliverySequenceNumber: number;
  sourceTimestamp: number;
  signingOfVaaTimestamp: number | undefined;
  deliveryInstruction: DeliveryInstruction;
  additionalMessageInformation: AdditionalMessageParsed[];
  targetChainStatus: {
    chain: ChainName;
    events: DeliveryTargetInfo[];
  };
  stringified?: string;
};

export type DeliveryArguments = {
  budget: BigNumber;
  deliveryInstruction: DeliveryInstruction;
  deliveryHash: string;
};

export async function manualDelivery(
  sourceChain: ChainName,
  sourceTransaction: string,
  infoRequest?: InfoRequestParams,
  getQuoteOnly?: boolean,
  overrides?: DeliveryOverrideArgs,
  signer?: ethers.Signer
): Promise<{ quote: BigNumber; targetChain: ChainName; txHash?: string }> {
  const info = await getWormholeRelayerInfo(
    sourceChain,
    sourceTransaction,
    infoRequest
  );
  const environment = infoRequest?.environment || "MAINNET";
  const sourceProvider =
    infoRequest?.sourceChainProvider ||
    getDefaultProvider(environment, sourceChain);
  const receipt = await sourceProvider.getTransactionReceipt(sourceTransaction);
  const wormholeRelayerAddress =
    infoRequest?.wormholeRelayerAddresses?.get(sourceChain) ||
    getWormholeRelayerAddress(sourceChain, environment);
  const response = await (
    await getWormscanInfo(
      environment,
      info.sourceChain,
      info.sourceDeliverySequenceNumber,
      wormholeRelayerAddress
    )
  ).json();

  const signedVaa = response.data.vaa;
  const signedVaaBuffer = Buffer.from(signedVaa, "base64");
  const result: { quote: BigNumber; targetChain: ChainName; txHash?: string } =
    {
      quote: deliveryBudget(info.deliveryInstruction, overrides),
      targetChain:
        CHAIN_ID_TO_NAME[info.deliveryInstruction.targetChainId as ChainId],
      txHash: undefined,
    };
  if (getQuoteOnly) {
    return result;
  } else {
    if (!signer) {
      throw new Error("no signer provided");
    }
    const deliveryReceipt = await deliver(
      signedVaaBuffer,
      signer,
      environment,
      overrides,
      sourceChain,
      receipt
    );
    result.txHash = deliveryReceipt.transactionHash;
    return result;
  }
}

export async function deliver(
  deliveryVaa: SignedVaa,
  signer: ethers.Signer,
  environment: Network = "MAINNET",
  overrides?: DeliveryOverrideArgs,
  sourceChain?: ChainName,
  sourceReceipt?: ethers.providers.TransactionReceipt
): Promise<ContractReceipt> {
  const { budget, deliveryInstruction, deliveryHash } =
    extractDeliveryArguments(deliveryVaa, overrides);

  const additionalMessages = await fetchAdditionalMessages(
    deliveryInstruction.messageKeys,
    environment,
    sourceChain,
    sourceReceipt
  );

  const wormholeRelayerAddress = getWormholeRelayerAddress(
    toChainName(deliveryInstruction.targetChainId as ChainId),
    environment
  );
  const wormholeRelayer = IWormholeRelayer__factory.connect(
    wormholeRelayerAddress,
    signer
  );
  const gasEstimate = await wormholeRelayer.estimateGas.deliver(
    additionalMessages,
    deliveryVaa,
    signer.getAddress(),
    overrides ? packOverrides(overrides) : new Uint8Array(),
    { value: budget }
  );
  const tx = await wormholeRelayer.deliver(
    additionalMessages,
    deliveryVaa,
    signer.getAddress(),
    overrides ? packOverrides(overrides) : new Uint8Array(),
    { value: budget, gasLimit: gasEstimate.mul(2) }
  );
  const rx = await tx.wait();
  return rx;
}

export function deliveryBudget(
  delivery: DeliveryInstruction,
  overrides?: DeliveryOverrideArgs
): BigNumber {
  const receiverValue = overrides?.newReceiverValue
    ? overrides.newReceiverValue
    : delivery.requestedReceiverValue.add(delivery.extraReceiverValue);
  const getMaxRefund = (encodedDeliveryInfo: Buffer) => {
    const [deliveryInfo] = parseEVMExecutionInfoV1(encodedDeliveryInfo, 0);
    return deliveryInfo.targetChainRefundPerGasUnused.mul(
      deliveryInfo.gasLimit
    );
  };
  const maxRefund = getMaxRefund(
    overrides?.newExecutionInfo
      ? overrides.newExecutionInfo
      : delivery.encodedExecutionInfo
  );
  return receiverValue.add(maxRefund);
}

export function extractDeliveryArguments(
  vaa: SignedVaa,
  overrides?: DeliveryOverrideArgs
): DeliveryArguments {
  const parsedVaa = parseVaa(vaa);

  const payloadType = parseWormholeRelayerPayloadType(parsedVaa.payload);
  if (payloadType !== RelayerPayloadId.Delivery) {
    throw new Error(
      `Expected delivery payload type, got ${RelayerPayloadId[payloadType]}`
    );
  }
  const deliveryInstruction = parseWormholeRelayerSend(parsedVaa.payload);
  const budget = deliveryBudget(deliveryInstruction, overrides);
  return {
    budget,
    deliveryInstruction: deliveryInstruction,
    deliveryHash: parsedVaa.hash.toString("hex"),
  };
}

export async function fetchAdditionalMessages(
  additionalMessageKeys: MessageKey[],
  environment: Network,
  sourceChain?: ChainName,
  sourceReceipt?: ethers.providers.TransactionReceipt
): Promise<(Uint8Array | Buffer)[]> {
  const messages = await Promise.all(
    additionalMessageKeys.map(async (messageKey) => {
      if (messageKey.keyType === 1) {
        const vaaKey = parseVaaKey(messageKey.key);
        const signedVaa = (
          await await (
            await getWormscanInfo(
              environment,
              CHAIN_ID_TO_NAME[vaaKey.chainId as ChainId],
              vaaKey.sequence.toNumber(),
              "0x" + vaaKey.emitterAddress.toString("hex")
            )
          ).json()
        ).data?.vaa;
        if (!signedVaa) {
          throw new Error(
            `No signed VAA available on WormScan for vaaKey ${JSON.stringify(
              vaaKey
            )}`
          );
        }
        return Buffer.from(signedVaa, "base64");
      } else if (messageKey.keyType === 2) {
        const cctpKey = parseCCTPKey(messageKey.key);
        if (!sourceReceipt)
          throw new Error(
            "No source receipt provided - needed to obtain CCTP message"
          );
        if (!environment)
          throw new Error(
            "No environment provided - needed to obtain CCTP message"
          );
        if (!sourceChain)
          throw new Error(
            "No source chain provided - needed to obtain CCTP message"
          );

        const response = await getCCTPMessageLogURL(
          cctpKey,
          sourceChain,
          sourceReceipt,
          environment
        );

        // Try to get attestation
        const attestationResponse = await fetch(response?.url || "");
        const attestationResponseJson = await attestationResponse.json();
        const attestation = attestationResponseJson.attestation;
        if (!attestation) {
          throw new Error(
            `Unable to get attestation from Circle, for cctp key ${JSON.stringify(
              cctpKey
            )}, message ${response?.message}`
          );
        }
        return Buffer.from(
          new ethers.utils.AbiCoder()
            .encode(["bytes", "bytes"], [response?.message || [], attestation])
            .substring(2),
          "hex"
        );
      } else {
        throw new Error(
          `Message key type unknown: ${messageKey.keyType} (messageKey ${messageKey.key})`
        );
      }
    })
  );
  return messages;
}
