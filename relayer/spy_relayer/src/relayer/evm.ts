import {
  getIsTransferCompletedEth,
  hexToUint8Array,
  redeemOnEth,
  redeemOnEthNative,
} from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { ethers } from "ethers";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger } from "../helpers/logHelper";

export function newProvider(
  url: string
): ethers.providers.WebSocketProvider | ethers.providers.JsonRpcProvider {
  // only support http(s), not ws(s) as the websocket constructor can blow up the entire process
  // it uses a nasty setTimeout(()=>{},0) so we are unable to cleanly catch its errors
  if (url.startsWith("http")) {
    return new ethers.providers.JsonRpcProvider(url);
  }
  throw new Error("url does not start with http/https!");
}

export async function relayEVM(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  checkOnly: boolean,
  walletPrivateKey: string
) {
  const logger = getScopedLogger(["relay", "evm", chainConfigInfo.chainName]);
  const signedVaaArray = hexToUint8Array(signedVAA);
  let provider = newProvider(chainConfigInfo.nodeUrl);
  const signer: Signer = new ethers.Wallet(walletPrivateKey, provider);

  if (unwrapNative) {
    logger.info(
      "Will redeem and unwrap using pubkey: %s",
      await signer.getAddress()
    );
  } else {
    logger.info("Will redeem using pubkey: %s", await signer.getAddress());
  }

  logger.debug("Checking to see if vaa has already been redeemed.");
  const alreadyRedeemed = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (alreadyRedeemed) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  logger.debug("Redeeming.");
  const receipt = unwrapNative
    ? await redeemOnEthNative(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray
      )
    : await redeemOnEth(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray
      );

  logger.debug("Checking to see if the transaction is complete.");

  const success = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }

  logger.info("success: %s tx hash: %s", success, receipt.transactionHash);
  return { redeemed: success, result: receipt };
}
