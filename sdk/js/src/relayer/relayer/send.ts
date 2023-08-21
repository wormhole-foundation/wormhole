import { ethers, BigNumber } from "ethers";
import { ethers_contracts } from "../..";
import {
  ChainId,
  ChainName,
  CHAINS,
  Network,
  tryNativeToHexString,
} from "../../utils";
import { getWormholeRelayerAddress } from "../consts";
import {
  MessageKeyStruct,
  VaaKeyStruct,
} from "../../ethers-contracts/IWormholeRelayer.sol/IWormholeRelayer";
import { encodeVaaKey } from "../structs";

export type SendOptionalParams = {
  environment?: Network;
  receiverValue?: ethers.BigNumberish;
  paymentForExtraReceiverValue?: ethers.BigNumberish;
  additionalMessages?: [
    {
      // Either specify the following fields (VaaKey)
      chainId?: ChainId;
      emitterAddress?: string;
      sequenceNumber?: ethers.BigNumberish;
      // Or specify a different message type to be relayed!
      keyType?: number;
      encodedKey?: string;
    }
  ];
  deliveryProviderAddress?: string;
  wormholeRelayerAddress?: string;
  consistencyLevel?: ethers.BigNumberish;
  refundChainId?: ChainId;
  refundAddress?: string;
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
  const sourceWormholeRelayer =
    ethers_contracts.IWormholeRelayer__factory.connect(
      wormholeRelayerAddress,
      signer
    );

  const refundLocationExists =
    sendOptionalParams?.refundChainId !== undefined &&
    sendOptionalParams?.refundAddress !== undefined;
  const defaultDeliveryProviderAddress =
    await sourceWormholeRelayer.getDefaultDeliveryProvider();

  // Using the most general 'send' function in IWormholeRelayer
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
    "sendToEvm(uint16,address,bytes,uint256,uint256,uint256,uint16,address,address,(uint8,bytes)[],uint8)"
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
    sendOptionalParams?.additionalMessages
      ? sendOptionalParams.additionalMessages.map(
          (additionalMessage): MessageKeyStruct => {
            if (additionalMessage.keyType) {
              if (!additionalMessage.encodedKey) {
                throw Error("No encoded key information provided!");
              }
              return {
                keyType: additionalMessage.keyType,
                encodedKey: additionalMessage.encodedKey,
              };
            } else {
              if (
                !additionalMessage.emitterAddress ||
                !additionalMessage.sequenceNumber
              ) {
                throw Error(
                  "No emitter address or sequence number information provided!"
                );
              }
              const key = {
                chainId: additionalMessage.chainId || sourceChainId,
                emitterAddress: Buffer.from(
                  tryNativeToHexString(
                    additionalMessage.emitterAddress,
                    "ethereum"
                  ),
                  "hex"
                ),
                sequence: BigNumber.from(additionalMessage.sequenceNumber || 0),
              };
              return {
                keyType: 1, // VAA KEY,
                encodedKey: encodeVaaKey(key),
              };
            }
          }
        )
      : [], // messageKeys
    sendOptionalParams?.consistencyLevel || 15, // consistencyLevel
    overrides
  );
  return tx;
}
