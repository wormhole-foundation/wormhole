import {
  hexToUint8Array,
  redeemOnEth,
  redeemOnEthNative,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { ChainConfigInfo } from "../configureEnv";

export async function relayEVM(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  request: any,
  response: any
) {
  const provider = new ethers.providers.WebSocketProvider(
    chainConfigInfo.nodeUrl
  );
  const signer = new ethers.Wallet(chainConfigInfo.walletPrivateKey, provider);
  const receipt = unwrapNative
    ? await redeemOnEthNative(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        hexToUint8Array(signedVAA)
      )
    : await redeemOnEth(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        hexToUint8Array(signedVAA)
      );
  provider.destroy();
  console.log("successfully redeemed on evm", receipt);
  response.status(200).json(receipt);
}
