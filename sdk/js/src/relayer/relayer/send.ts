import { ethers, BigNumber } from "ethers";
import { ethers_contracts } from "../..";
import { VaaKeyStruct } from "../../ethers-relayer-contracts/MockRelayerIntegration";
import { IWormholeRelayer__factory } from "../../ethers-relayer-contracts";
import {
  ChainId,
  ChainName,
  CHAINS,
  Network,
  tryNativeToHexString,
} from "../../utils";
import { getWormholeRelayerAddress } from "../consts";

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
  deliveryProviderAddress?: string;
  wormholeRelayerAddress?: string;
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
  gasLimit: BigNumber | number,
  overrides?: ethers.PayableOverrides,
  sendOptionalParams?: SendOptionalParams
): Promise<ethers.providers.TransactionResponse> {
  const sourceChainId = CHAINS[sourceChain];
  const targetChainId = CHAINS[targetChain];

  const environment = sendOptionalParams?.environment || "MAINNET";
  const wormholeRelayerAddress =
    sendOptionalParams?.wormholeRelayerAddress ||
    getWormholeRelayerAddress(sourceChain, environment);
  const sourceWormholeRelayer = IWormholeRelayer__factory.connect(
    wormholeRelayerAddress,
    signer
  );

  const refundLocationExists =
    sendOptionalParams?.refundChainId !== undefined &&
    sendOptionalParams?.refundAddress !== undefined;
  const defaultDeliveryProviderAddress =
    await sourceWormholeRelayer.getDefaultDeliveryProvider();

  // Using the most general 'send' function in IWormholeRelayer
  // Inputs:
  // targetChainId, targetAddress, refundChainId, refundAddress, maxTransactionFee, receiverValue, payload, vaaKeys,
  // consistencyLevel, deliveryProviderAddress, relayParameters
  const [deliveryPrice]: [BigNumber, BigNumber] = await sourceWormholeRelayer[
    "quoteEVMDeliveryPrice(uint16,uint256,uint256,address)"
  ](
    targetChainId,
    sendOptionalParams?.receiverValue || 0,
    gasLimit,
    sendOptionalParams?.deliveryProviderAddress ||
      defaultDeliveryProviderAddress
  );
  const value = await (overrides?.value || 0);
  const totalPrice = deliveryPrice.add(
    sendOptionalParams?.paymentForExtraReceiverValue || 0
  );
  if (!totalPrice.eq(value)) {
    throw new Error(
      `Expected a payment of ${totalPrice.toString()} wei; received ${value.toString()} wei`
    );
  }
  const tx = sourceWormholeRelayer[
    "sendToEvm(uint16,address,bytes,uint256,uint256,uint256,uint16,address,address,(uint16,bytes32,uint64)[],uint8)"
  ](
    targetChainId, // targetChainId
    targetAddress, // targetAddress
    payload,
    sendOptionalParams?.receiverValue || 0, // receiverValue
    sendOptionalParams?.paymentForExtraReceiverValue || 0, // payment for extra receiverValue
    gasLimit,
    (refundLocationExists && sendOptionalParams?.refundChainId) ||
      sourceChainId, // refundChainId
    (refundLocationExists &&
      sendOptionalParams?.refundAddress &&
      sendOptionalParams?.refundAddress) ||
      signer.getAddress(), // refundAddress
    sendOptionalParams?.deliveryProviderAddress ||
      defaultDeliveryProviderAddress, // deliveryProviderAddress
    sendOptionalParams?.additionalVaas
      ? sendOptionalParams.additionalVaas.map(
          (additionalVaa): VaaKeyStruct => ({
            chainId: additionalVaa.chainId || sourceChainId,
            emitterAddress: Buffer.from(
              tryNativeToHexString(additionalVaa.emitterAddress, "ethereum"),
              "hex"
            ),
            sequence: BigNumber.from(additionalVaa.sequenceNumber || 0),
          })
        )
      : [], // vaaKeys
    sendOptionalParams?.consistencyLevel || 15, // consistencyLevel
    overrides
  );
  return tx;
}
