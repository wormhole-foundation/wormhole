import { BigNumber, ethers, ContractReceipt } from "ethers";
import { IWormholeRelayer__factory } from "../../ethers-contracts";
import { ChainName, toChainName, ChainId, Network } from "../../utils";
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
  VaaKey,
} from "../structs";
import { DeliveryTargetInfo } from "./helpers";
import { getSignedVAAWithRetry } from "../../rpc";

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

export type DeliveryArguments = {
  budget: BigNumber;
  deliveryInstruction: DeliveryInstruction;
  deliveryHash: string;
};

export async function deliver(
  deliveryVaa: SignedVaa,
  signer: ethers.Signer,
  wormholeRPCs: string | string[],
  environment: Network = "MAINNET",
  overrides?: DeliveryOverrideArgs
): Promise<ContractReceipt> {
  const { budget, deliveryInstruction, deliveryHash } =
    extractDeliveryArguments(deliveryVaa, overrides);

  const additionalVaas = await fetchAdditionalVaas(
    wormholeRPCs,
    deliveryInstruction.vaaKeys
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
    additionalVaas,
    deliveryVaa,
    signer.getAddress(),
    overrides ? packOverrides(overrides) : new Uint8Array(),
    { value: budget }
  );
  const tx = await wormholeRelayer.deliver(
    additionalVaas,
    deliveryVaa,
    signer.getAddress(),
    overrides ? packOverrides(overrides) : new Uint8Array(),
    { value: budget, gasLimit: gasEstimate.mul(2) }
  );
  const rx = await tx.wait();
  console.log(`Delivered ${deliveryHash} on ${rx.blockNumber}`);
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

export async function fetchAdditionalVaas(
  wormholeRPCs: string | string[],
  additionalVaaKeys: VaaKey[]
): Promise<SignedVaa[]> {
  const rpcs = typeof wormholeRPCs === "string" ? [wormholeRPCs] : wormholeRPCs;
  const vaas = await Promise.all(
    additionalVaaKeys.map(async (vaaKey) =>
      getSignedVAAWithRetry(
        rpcs,
        vaaKey.chainId as ChainId,
        vaaKey.emitterAddress.toString("hex"),
        vaaKey.sequence.toBigInt().toString()
      )
    )
  );
  return vaas.map((vaa) => vaa.vaaBytes);
}
