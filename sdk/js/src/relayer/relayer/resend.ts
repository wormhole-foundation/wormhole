import { ethers, BigNumber } from "ethers";
import { ChainName, CHAINS, Network } from "../../utils";
import { parseVaa } from "../../vaa";
import { getWormholeRelayer } from "../consts";
import {
  VaaKey,
  parseWormholeRelayerSend,
  parseEVMExecutionInfoV1,
} from "../structs";
import { vaaKeyToVaaKeyStruct, getDeliveryProvider, getVAA } from "./helpers";

export async function resendRaw(
  signer: ethers.Signer,
  sourceChain: ChainName,
  targetChain: ChainName,
  environment: Network,
  vaaKey: VaaKey,
  newGasLimit: BigNumber | number,
  newReceiverValue: BigNumber | number,
  deliveryProviderAddress: string,
  overrides?: ethers.PayableOverrides
) {
  const provider = signer.provider;

  if (!provider) throw Error("No provider on signer");

  const wormholeRelayer = getWormholeRelayer(sourceChain, environment, signer);

  return wormholeRelayer.resendToEvm(
    vaaKeyToVaaKeyStruct(vaaKey),
    CHAINS[targetChain],
    newReceiverValue,
    newGasLimit,
    deliveryProviderAddress,
    overrides
  );
}

export async function resend(
  signer: ethers.Signer,
  sourceChain: ChainName,
  targetChain: ChainName,
  environment: Network,
  vaaKey: VaaKey,
  newGasLimit: BigNumber | number,
  newReceiverValue: BigNumber | number,
  deliveryProviderAddress: string,
  wormholeRPCs: string[],
  overrides: ethers.PayableOverrides,
  isNode?: boolean
) {
  const targetChainId = CHAINS[targetChain];
  const originalVAA = await getVAA(wormholeRPCs, vaaKey, isNode);

  if (!originalVAA) throw Error("orignal VAA not found");

  const originalVAAparsed = parseWormholeRelayerSend(
    parseVaa(Buffer.from(originalVAA)).payload
  );
  if (!originalVAAparsed) throw Error("orignal VAA not a valid delivery VAA.");

  const [originalExecutionInfo] = parseEVMExecutionInfoV1(
    originalVAAparsed.encodedExecutionInfo,
    0
  );
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

  const wormholeRelayer = getWormholeRelayer(sourceChain, environment, signer);
  const deliveryProvider = getDeliveryProvider(
    deliveryProviderAddress,
    signer.provider!
  );

  const [deliveryPrice, refundPerUnitGas]: [BigNumber, BigNumber] =
    await wormholeRelayer[
      "quoteEVMDeliveryPrice(uint16,uint256,uint256,address)"
    ](
      targetChainId,
      newReceiverValue || 0,
      newGasLimit,
      deliveryProviderAddress
    );
  const value = await (overrides?.value || 0);
  if (!deliveryPrice.eq(value)) {
    throw new Error(
      `Expected a payment of ${deliveryPrice.toString()} wei; received ${value.toString()} wei`
    );
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
    deliveryProviderAddress,
    overrides
  );
}
